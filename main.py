import logging as log
from typing import *
import multiprocessing as mp
import os
import math

import src.proc as proc
import src.path as path
import src.ffmpeg as ffmpeg

log.basicConfig(
    format="[%(levelname)s][%(asctime)s.%(msecs)03d][%(filename)s:%(lineno)s,%(funcName)s()] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    level=log.INFO)


def ffmpeg_segment(fp_in: str, ext_out: str):
    with proc.NestablePool(math.ceil(mp.cpu_count() / 4)) as pool:
        async_audio = pool.map_async(ffmpeg.encode_audio_only, [fp_in])

        # split into video segments
        fps_video_split = ffmpeg.split_video(
            fp_in, math.ceil(mp.cpu_count() / 4))
        log.info(fps_video_split)

        # transcode video segments
        fps_video_encode = pool.starmap(ffmpeg.encode_video_only, list(
            zip(fps_video_split, [ext_out for _ in range(len(fps_video_split))])))
        log.info(fps_video_encode)

        # after transcoding, remove old video segments
        for fp in fps_video_split:
            os.remove(fp)

        # concat video segments
        fp_video = ffmpeg.concat_files(fps_video_encode, ext_out)
        log.info(fp_video)

        # after concat, remove transcoded video segments
        for fp in fps_video_encode:
            os.remove(fp)

        # wait until audio transcoding is finished
        fp_audio = async_audio.get()[0]

        # merge video and audio
        fp_out = ffmpeg.mux_video_audio(fp_video, fp_audio)
        log.info(fp_out)

        # remove old video and audio file
        os.remove(fp_video)
        os.remove(fp_audio)


def main():
    """
    with proc.NestablePool(2) as pool:
        pool.map(ffmpeg_segment, path.listdir('C:/Users/heeyong/Desktop/test'))
    """
    for fp in path.listdir('C:/Users/heeyong/Desktop/test'):
        ffmpeg_segment(fp, '.webm')


if __name__ == '__main__':
    main()
