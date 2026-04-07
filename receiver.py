import pika
import spotipy
from spotipy.oauth2 import SpotifyOAuth
from dotenv import load_dotenv

load_dotenv()

auth_manager = SpotifyOAuth(
    scope="user-modify-playback-state user-read-playback-state",
    open_browser=True,
)

# Force auth flow now, before the RabbitMQ consumer starts.
# This ensures the interactive prompt happens at a clean stdin, not mid-callback.
auth_manager.get_access_token(as_dict=False)

sp = spotipy.Spotify(auth_manager=auth_manager)

def seek_forward():
    playback = sp.current_playback()
    if playback and playback.get("progress_ms") is not None:
        current_ms = playback["progress_ms"]
        duration_ms = playback["item"]["duration_ms"]
        new_pos = min(current_ms + 10_000, duration_ms)
        sp.seek_track(new_pos)


def seek_backward():
    playback = sp.current_playback()
    if playback and playback.get("progress_ms") is not None:
        current_ms = playback["progress_ms"]
        new_pos = max(current_ms - 10_000, 0)
        sp.seek_track(new_pos)


ACTIONS = {
    "PLAY":           lambda: sp.start_playback(),
    "PAUSE":          lambda: sp.pause_playback(),
    "SKIP_FORWARD":   lambda: sp.next_track(),
    "SKIP_BACKWARD":  lambda: sp.previous_track(),
    "SEEK_FORWARD":   seek_forward,
    "SEEK_BACKWARD":  seek_backward,
}


def on_message(ch, method, properties, body):
    action = body.decode()
    print(f"Received: {action}")
    fn = ACTIONS.get(action)
    if fn:
        try:
            fn()
            print(f"  -> OK")
        except spotipy.SpotifyException as e:
            print(f"  -> Spotify error: {e}")
    ch.basic_ack(delivery_tag=method.delivery_tag)


connection = pika.BlockingConnection(pika.ConnectionParameters("localhost", heartbeat=15))
channel = connection.channel()
channel.queue_declare(queue="spotify_actions", durable=True, arguments={"x-max-length": 1})
channel.basic_qos(prefetch_count=1)
channel.basic_consume(queue="spotify_actions", on_message_callback=on_message)

print("Waiting for messages... (Ctrl+C to stop)")
try:
    channel.start_consuming()
except KeyboardInterrupt:
    channel.stop_consuming()

connection.close()
