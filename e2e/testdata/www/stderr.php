<?php

// php-fpm catches the stderr of the worker and writes it to the error log.

$stderr = fopen('php://stderr', 'wb');
fwrite($stderr, (isset($_GET['message']) ? $_GET['message'] : 'worker stderr line') . "\n");
fclose($stderr);

echo 'ok';
