# Spotify Gesture Controller — Project Plan

A real-time, event-driven system to control Spotify using hand gestures via MediaPipe and RabbitMQ.

---

## System Architecture

Three independent Python services communicate through a single RabbitMQ queue:

```
[sender.py]  →  RabbitMQ (spotify_actions queue)  →  [receiver.py]
 Camera + MediaPipe                                    Spotipy API
 (threaded capture)
```

**Services:**
- **sender.py** — Captures webcam frames in a background thread, runs MediaPipe gesture detection, publishes action strings to RabbitMQ.
- **RabbitMQ** — Durable message queue. Decouples vision from API so a Spotify hiccup never drops a frame.
- **receiver.py** — Consumes messages and calls the Spotify Web API via Spotipy.

---

## Supported Actions

| Action | Spotify Call |
|---|---|
| `PLAY` | `spotify.start_playback()` |
| `PAUSE` | `spotify.pause_playback()` |
| `SKIP_FORWARD` | `spotify.next_track()` |
| `SKIP_BACKWARD` | `spotify.previous_track()` |

---

## Phase 1 — Environment Setup

### RabbitMQ (Docker)

```bash
docker run -d --name rabbitmq \
  -p 5672:5672 -p 15672:15672 \
  rabbitmq:3-management
```

Management UI: http://localhost:15672 (guest / guest)

### Python Dependencies

```bash
pip install opencv-python mediapipe pika spotipy python-dotenv
```

### `.env` file

```
SPOTIPY_CLIENT_ID=your_client_id
SPOTIPY_CLIENT_SECRET=your_client_secret
SPOTIPY_REDIRECT_URI=http://localhost:3000/callback
```

---

## Phase 2 — Spotify Receiver (`receiver.py`)

Build and test this first — no camera required.

### Spotify App Setup

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard).
2. Create an app. Set Redirect URI to `http://localhost:3000/callback`.
3. Copy Client ID and Secret into `.env`.

### Key Implementation Points

**Auth** — Use `SpotifyOAuth` with the required scopes. Spotipy handles token refresh automatically; no manual refresh logic needed.

```python
import spotipy
from spotipy.oauth2 import SpotifyOAuth

sp = spotipy.Spotify(auth_manager=SpotifyOAuth(
    scope="user-modify-playback-state user-read-playback-state",
    open_browser=True
))
```

**Queue declaration** — Declare the queue as `durable=True` and consume with `auto_ack=False`. Manually `ack` after a successful API call so failed messages requeue.

```python
channel.queue_declare(queue="spotify_actions", durable=True)
channel.basic_qos(prefetch_count=1)
channel.basic_consume(queue="spotify_actions", on_message_callback=on_message)
```

**Message handler** — Map action strings to Spotify calls with error handling:

```python
ACTIONS = {
    "PLAY":          lambda: sp.start_playback(),
    "PAUSE":         lambda: sp.pause_playback(),
    "SKIP_FORWARD":  lambda: sp.next_track(),
    "SKIP_BACKWARD": lambda: sp.previous_track(),
}

def on_message(ch, method, properties, body):
    action = body.decode()
    fn = ACTIONS.get(action)
    if fn:
        try:
            fn()
        except spotipy.SpotifyException as e:
            print(f"Spotify error: {e}")
    ch.basic_ack(delivery_tag=method.delivery_tag)
```

**Test it standalone** before writing a single line of camera code.

---

## Phase 3 — MediaPipe Sender (`sender.py`)

### Gesture Mapping (starting point — tune to your hand)

| Gesture | Action |
|---|---|
| Open palm facing camera | `PLAY` |
| Closed fist | `PAUSE` |
| Thumb pointing right | `SKIP_FORWARD` |
| Thumb pointing left | `SKIP_BACKWARD` |

MediaPipe's `GestureRecognizer` task (introduced in MediaPipe 0.10) returns named gestures like `"Open_Palm"`, `"Closed_Fist"`, `"Thumb_Up"` directly — prefer this over manual landmark math.

### Threading for Camera I/O

OpenCV's `cap.read()` blocks until a new frame arrives. On a 30 fps camera that's ~33 ms of blocking per frame. Run capture in a daemon thread so the gesture detection loop never stalls waiting on the camera:

```python
import threading, queue, cv2

frame_queue = queue.Queue(maxsize=2)  # small buffer — we want the latest frame

def capture_thread(cap, q):
    while True:
        ret, frame = cap.read()
        if not ret:
            break
        if not q.full():
            q.put(frame)

cap = cv2.VideoCapture(0)
t = threading.Thread(target=capture_thread, args=(cap, frame_queue), daemon=True)
t.start()

# Main loop reads latest frame without blocking
while True:
    if not frame_queue.empty():
        frame = frame_queue.get()
        # run MediaPipe detection here
```

`daemon=True` means the thread exits automatically when the main process ends.

### Debouncing

A gesture held for 1 second appears in ~30 consecutive frames. Send only once per gesture change, with a minimum cooldown between sends:

```python
import time

COOLDOWN = 1.5  # seconds
last_action = None
last_sent = 0.0

def maybe_publish(action, channel):
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
        properties=pika.BasicProperties(delivery_mode=2)  # persistent
    )
    print(f"Sent: {action}")
```

`delivery_mode=2` makes messages survive a RabbitMQ restart.

---

## Phase 4 — Integration & Testing

**Start order:**

```bash
# Terminal 1 — already running from Phase 1
docker start rabbitmq

# Terminal 2
python receiver.py   # opens browser for Spotify login on first run

# Terminal 3 — once receiver prints "Waiting for messages..."
python sender.py
```

**Verify in the RabbitMQ UI** at http://localhost:15672 → Queues → `spotify_actions`:
- **Ready** should sit near 0 (messages being consumed promptly).
- **Unacked** spikes to 1 during processing, then drops.

**Tune as you go:**
- Adjust `COOLDOWN` if gestures feel sluggish or fire too often.
- Raise the MediaPipe confidence threshold if false positives occur.
- Swap `queue.Queue(maxsize=2)` to `maxsize=1` to always process the absolute latest frame.

---

## Project Structure

```
spotify-gesture/
├── .env
├── receiver.py
├── sender.py
└── requirements.txt
```

Keep it flat. Two files is the right scope for this project — resist splitting into packages until there's a clear reason.

---

## Quick Reference

| Component | Default |
|---|---|
| RabbitMQ AMQP port | 5672 |
| RabbitMQ UI | http://localhost:15672 |
| Spotify redirect URI | http://localhost:3000/callback |
| Queue name | `spotify_actions` |
| Debounce cooldown | 1.5 s |
| Camera thread buffer | 2 frames |
