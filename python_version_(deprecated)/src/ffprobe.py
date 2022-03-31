from importlib.resources import path
import json
from . import path
import shlex
from . import proc


def stream_info_json(input_file_path: str) -> json:
    input_file_path = path.sanitize(input_file_path)
    arg = shlex.split(
        'ffprobe -v quiet -print_format json -show_streams') + [input_file_path]
    out, _ = proc.run(arg)
    return json.loads(out)


def video_time(input_file_path: str) -> float:
    input_file_path = path.sanitize(input_file_path)
    arg = shlex.split(
        'ffprobe -v error -select_streams v:0 -show_entries format=duration -of default=noprint_wrappers=1:nokey=1') + [input_file_path]
    out, _ = proc.run(arg)
    return float(out)


def video_frame(input_file_path: str) -> int:
    input_file_path = path.sanitize(input_file_path)
    arg = shlex.split(
        'ffprobe -v error -select_streams v:0 -count_packets -show_entries stream=nb_read_packets -of csv=p=0') + [input_file_path]
    out, _ = proc.run(arg)
    return float(out)
