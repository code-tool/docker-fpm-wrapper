<?php

declare(strict_types=1);

// Writes lines to the log socket and to the log pipe of the wrapper.

function writeToSocket(string $data): void
{
    $socket = stream_socket_client(getenv('FPM_WRAPPER_SOCK'), $errorCode, $errorMessage, 1.0);
    if ($socket === false) {
        throw new RuntimeException("cannot open the wrapper socket: $errorMessage");
    }

    fwrite($socket, $data);
    fclose($socket);
}

function writeToPipe(string $data): void
{
    $pipe = fopen(getenv('FPM_WRAPPER_PIPE'), 'ab');
    if ($pipe === false) {
        throw new RuntimeException('cannot open the wrapper pipe');
    }

    fwrite($pipe, $data);
    fclose($pipe);
}

writeToSocket("socket line\n");
writeToSocket("socket crlf line\r\n");
writeToSocket('socket line without newline');
// The wrapper drops a line that is longer than --line-buffer-size and keeps the next line.
writeToSocket(str_repeat('x', 20 * 1024) . "\nsocket line after long line\n");

writeToPipe("pipe line\n");

echo 'ok';
