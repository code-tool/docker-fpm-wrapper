<?php

declare(strict_types=1);

// The request is longer than request_slowlog_timeout, so php-fpm writes the trace to the slowlog.

function slowOuter(): void
{
    slowInner();
}

function slowInner(): void
{
    sleep(2);
}

slowOuter();

echo 'ok';
