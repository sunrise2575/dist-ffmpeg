from typing import List
import multiprocessing
import multiprocessing.pool
import subprocess
import logging


def run(arg: List[int]):
    process = subprocess.Popen(arg,
                               shell=True,
                               stdin=subprocess.PIPE,
                               stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE,
                               encoding='CP949',  # for windows
                               universal_newlines=True)
    (stdout, stderr) = process.communicate()
    if len(stdout) > 0:
        logging.info(stdout)
    if len(stderr) > 0:
        logging.error(stderr)
    return stdout, stderr


class NoDaemonProcess(multiprocessing.Process):
    @property
    def daemon(self):
        return False

    @daemon.setter
    def daemon(self, value):
        pass


class NoDaemonContext(type(multiprocessing.get_context())):
    Process = NoDaemonProcess


class NestablePool(multiprocessing.pool.Pool):
    def __init__(self, *args, **kwargs):
        kwargs['context'] = NoDaemonContext()
        super().__init__(*args, **kwargs)
