from msilib.schema import Error
import shlex
import math
import os
import re
from . import path
from . import proc
from . import ffprobe

FFMPEG_COMMON_INPUT_ARG = 'ffmpeg -hide_banner -loglevel warning -avoid_negative_ts 1 -analyzeduration 2147483647 -probesize 2147483647 -y '
FFMPEG_COMMON_OUTPUT_ARG = ' -max_muxing_queue_size 4096 '


def split_video(fp_in: str, expected_file_count: int) -> str:
    fp_in = path.sanitize(fp_in)
    dir, name, ext = path.split(fp_in)

    # plan for splitting video
    time = ffprobe.video_time(fp_in)
    unit_time = max(20, math.ceil(float(time) / float(expected_file_count)))
    expected_file_count = math.ceil(time / unit_time)

    # split video
    arg = shlex.split(FFMPEG_COMMON_INPUT_ARG + '-i')+[fp_in]
    arg += shlex.split('-f segment -segment_time') + [str(unit_time)]
    arg += shlex.split(FFMPEG_COMMON_OUTPUT_ARG+'-reset_timestamps 1 -c:v copy -map 0:v:0') + \
        [path.join(dir, '.'+name+'_video_%d', ext)]
    proc.run(arg)

    result = [path.join(dir, '.'+name+'_video_'+str(i), ext)
              for i in range(expected_file_count)]
    for fp in list(result):
        if not (os.path.exists(fp) and os.path.isfile(fp)):
            result.remove(fp)

    return result


def encode_audio_only(fp_in: str) -> str:
    fp_in = path.sanitize(fp_in)
    dir, name, _ = path.split(fp_in)
    outfile = path.join(dir, '.'+name+'_audio', '.ogg')

    arg = shlex.split(FFMPEG_COMMON_INPUT_ARG + '-i')+[fp_in]
    arg += shlex.split(FFMPEG_COMMON_OUTPUT_ARG +
                       '-c:a libopus -b:a 128k -map 0:a:0?')+[outfile]
    proc.run(arg)

    return outfile


def encode_video_only(fp_in: str, ext_out: str) -> str:
    dir, name, _ = path.split(fp_in)
    outfile = path.join(dir, name + '_encoded', ext_out)

    # transcode video
    arg = shlex.split(
        FFMPEG_COMMON_INPUT_ARG + '-threads 0 -i')+[fp_in]
    arg += shlex.split(
        FFMPEG_COMMON_OUTPUT_ARG+'-c:v libvpx-vp9 -b:v 0 -pix_fmt:v yuv420p -cpu-used:v 4 -crf:v 27') + [outfile]
    proc.run(arg)

    return outfile


# in progress
def concat_files(fps_in: str, ext_out: str) -> str:
    if len(fps_in) == 0:
        raise Error

    dir, common_name, ext = path.split(fps_in[0])
    for fp in fps_in:
        _dir, _, _ext = path.split(fp)
        if not (_dir == dir and _ext == ext):
            raise Error

    common_name = re.sub('_[0-9]*_encoded', '', common_name)
    fp_text = path.join(dir, common_name, '.txt')

    fp_out = path.join(dir, common_name, ext_out)

    with open(fp_text, 'w') as f_text:
        f_text.writelines(["file '" + fp + "'\n" for fp in fps_in])

    # concat video
    arg = shlex.split(
        FFMPEG_COMMON_INPUT_ARG + '-f concat -safe 0 -i')+[fp_text]
    arg += shlex.split(FFMPEG_COMMON_OUTPUT_ARG+'-c:v copy') + [fp_out]
    proc.run(arg)

    os.remove(fp_text)

    return fp_out


def mux_video_audio(fp_in_video: str, fp_in_audio: str) -> str:
    video_dir, video_name, video_ext = path.split(fp_in_video)
    audio_dir, _, _ = path.split(fp_in_video)
    if not (video_dir == audio_dir):
        raise Error

    out_name = re.sub('_(video|audio)$', '', video_name)
    out_name = re.sub('^\.', '', out_name)

    fp_out = path.join(video_dir, out_name, video_ext)

    # concat video
    arg = shlex.split(FFMPEG_COMMON_INPUT_ARG)
    arg += ['-i', fp_in_video]
    arg += ['-i', fp_in_audio]
    arg += shlex.split(FFMPEG_COMMON_OUTPUT_ARG +
                       '-c:v copy -c:a copy -map 0:v:0 -map 1:a:0') + [fp_out]
    proc.run(arg)

    return fp_out
