import math
import os
from threading import Thread, Lock
import mediapipe as mp
# from mediapipe.tasks.python import vision
import cv2
import time
from collections import deque
import pika
from dotenv import load_dotenv

load_dotenv()

NUM_HANDS = 1
HAND_MODEL_CONFIDENCE=0.5


class ThreadStream:
    def __init__(self, src=0):
        self.stream = cv2.VideoCapture(src)
        (self.has_frame, self.frame) = self.stream.read()
        self.stopped = False

    def start(self):
        Thread(target=self.update, args=()).start()

    def update(self):
        while True:
            if self.stopped:
                return
            (self.has_frame, self.frame) = self.stream.read()

    def stop(self):
        self.stopped = True

    def getFrame(self):
        return self.frame, int(time.time() * 1000)


# Gesture name → Spotify action
GESTURE_MAP = {
    "Open_Palm":   "PLAY",
    "Closed_Fist": "PAUSE",
    "Thumb_Up":    "SKIP_FORWARD",
    "Thumb_Down":  "SKIP_BACKWARD",
}

# Landmark connections for drawing the hand skeleton
LANDMARK_CONNECTIONS = [
    (0, 1), (1, 2), (2, 3), (3, 4),
    (0, 5), (5, 6), (6, 7), (7, 8),
    (0, 9), (9, 10), (10, 11), (11, 12),
    (0, 13), (13, 14), (14, 15), (15, 16),
    (0, 17), (17, 18), (18, 19), (19, 20),
]

# RabbitMQ
connection = pika.BlockingConnection(pika.ConnectionParameters("localhost", heartbeat=15))
channel = connection.channel()
channel.queue_declare(queue="spotify_actions", durable=True, arguments={"x-max-length": 1})

# Debounce state
COOLDOWN = 1.5
last_action = None
last_sent = 0.0


def maybe_publish(action):
    global last_action, last_sent
    now = time.monotonic()
    if action == last_action and (now - last_sent) < COOLDOWN:
        return
    last_action = action
    last_sent = now
    channel.basic_publish(
        exchange="",
        routing_key="spotify_actions",
        body=action,
        properties=pika.BasicProperties(delivery_mode=1),
    )
    print(f"Sent: {action}")


# MediaPipe hand landmark indices (fingertip / PIP / MCP) for index→pinky
_FINGER_TIP  = [8, 12, 16, 20]
_FINGER_PIP  = [6, 10, 14, 18]
_FINGER_MCP  = [5,  9, 13, 17]


def _angle_from_vertical(lms, tip_idx, mcp_idx):
    """Angle (degrees) between the MCP→TIP vector and straight up (0, -1)."""
    dx = lms[tip_idx].x - lms[mcp_idx].x
    dy = lms[tip_idx].y - lms[mcp_idx].y   # y increases downward in image coords
    return abs(math.degrees(math.atan2(dx, -dy)))


def detect_seek_gesture(lms):
    """
    Returns 'SEEK_FORWARD', 'SEEK_BACKWARD', or None.

    Uses angles to decide whether a finger is pointing up (MCP→TIP angle from
    vertical < 50°) and y-coordinates to decide whether a finger is curled
    (tip.y > pip.y).

    Two fingers up  (index + middle, ring + pinky curled) → SEEK_FORWARD
    Three fingers up (index + middle + ring, pinky curled) → SEEK_BACKWARD
    """
    UP_ANGLE = 50   # degrees from vertical to count as "pointing up"

    fingers_up   = [_angle_from_vertical(lms, _FINGER_TIP[i], _FINGER_MCP[i]) < UP_ANGLE
                    for i in range(4)]
    fingers_curl = [lms[_FINGER_TIP[i]].y > lms[_FINGER_PIP[i]].y
                    for i in range(4)]   # True when finger is curled

    index, middle, ring, pinky = fingers_up
    ring_curl, pinky_curl = fingers_curl[2], fingers_curl[3]

    if index and middle and ring_curl and pinky_curl:
        return "SEEK_FORWARD"
    if index and middle and ring and pinky_curl:
        return "SEEK_BACKWARD"
    return None


# MediaPipe GestureRecognizer setup
model_path = os.path.join(os.getcwd(), "gesture_recognizer.task")

BaseOptions = mp.tasks.BaseOptions
GestureRecognizer = mp.tasks.vision.GestureRecognizer
GestureRecognizerOptions = mp.tasks.vision.GestureRecognizerOptions
GestureRecognizerResult = mp.tasks.vision.GestureRecognizerResult
VisionRunningMode = mp.tasks.vision.RunningMode

gesture_res = [None]
gesture_lock = Lock()


def callback(result: GestureRecognizerResult, output_image: mp.Image, timestamp_ms: int):
    with gesture_lock:
        gesture_res[0] = result


options = GestureRecognizerOptions(
    base_options=BaseOptions(model_asset_path=model_path),
    running_mode=VisionRunningMode.LIVE_STREAM,
    result_callback=callback,
    num_hands=NUM_HANDS,
    min_hand_presence_confidence=HAND_MODEL_CONFIDENCE,
)

WIN_NAME = "Spotify Gesture Controller"
cv2.namedWindow(WIN_NAME, cv2.WINDOW_NORMAL)

ts = ThreadStream()
ts.start()

frame_times = deque(maxlen=30)
measured_fps = -1.0

with GestureRecognizer.create_from_options(options) as recognizer:
    alive = True
    while alive:
        frame, frame_timestamp_ms = ts.getFrame()

        if frame is None:
            break

        frame_times.append(time.perf_counter())
        if len(frame_times) == 30:
            measured_fps = (len(frame_times) - 1) / (frame_times[-1] - frame_times[0])

        mp_image = mp.Image(image_format=mp.ImageFormat.SRGB, data=frame)
        recognizer.recognize_async(mp_image, frame_timestamp_ms)

        h, w = frame.shape[:2]

        with gesture_lock:
            result = gesture_res[0]

        gesture_label = "None"
        if result is not None and result.gestures:
            gesture_name = result.gestures[0][0].category_name
            gesture_label = gesture_name

            # Draw hand skeleton
            if result.hand_landmarks:
                lms = result.hand_landmarks[0]
                for lndmark in lms:
                    cx, cy = int(w * lndmark.x), int(h * lndmark.y)
                    cv2.circle(frame, (cx, cy), 6, (0, 255, 0), -1)
                for a, b in LANDMARK_CONNECTIONS:
                    ax, ay = int(w * lms[a].x), int(h * lms[a].y)
                    bx, by = int(w * lms[b].x), int(h * lms[b].y)
                    cv2.line(frame, (ax, ay), (bx, by), (0, 200, 0), 2)

            seek_action = None
            if result.hand_landmarks:
                seek_action = detect_seek_gesture(result.hand_landmarks[0])

            if seek_action:
                maybe_publish(seek_action)
            else:
                action = GESTURE_MAP.get(gesture_name)
                if action:
                    maybe_publish(action)

        cv2.putText(frame, f"FPS: {measured_fps:.1f}", (10, 30), cv2.FONT_HERSHEY_SIMPLEX, 1, (0, 255, 0), 2)
        cv2.putText(frame, f"Action: {last_action or 'None'}", (10, 65), cv2.FONT_HERSHEY_SIMPLEX, 1, (255, 255, 0), 2)
        cv2.putText(frame, f"Gesture: {gesture_label}", (10, 100), cv2.FONT_HERSHEY_SIMPLEX, 1, (0, 100, 255), 2)
        cv2.imshow(WIN_NAME, frame)

        if cv2.waitKey(1) == 27:  # ESC to quit
            ts.stop()
            alive = False

connection.close()
cv2.destroyAllWindows()
