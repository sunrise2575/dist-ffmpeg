import os
from typing import Tuple


def listdir(dir: str):
    for dirpath, _, filenames in os.walk(dir):
        for f in filenames:
            yield os.path.abspath(os.path.join(dirpath, f))


def split(path: str) -> Tuple[str, str, str]:
    path = os.path.abspath(path)
    left, ext = os.path.splitext(path)
    dir = os.path.dirname(left)
    name = os.path.basename(left)
    return dir, name, ext


def join(dir: str, name: str, ext: str) -> str:
    return os.path.join(dir, name+ext)


def sanitize(filepath: str) -> str:
    filepath = os.path.abspath(filepath)
    if not os.path.exists(filepath):
        raise FileNotFoundError
    if os.path.isdir(filepath):
        raise IsADirectoryError
    return filepath
