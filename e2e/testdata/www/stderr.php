<?php

declare(strict_types=1);

// php-fpm catches the stderr of the worker and writes it to the error log.

$stderr = fopen('php://stderr', 'wb');
fwrite($stderr, ($_GET['message'] ?? 'worker stderr line') . "\n");
fclose($stderr);

echo 'ok';
