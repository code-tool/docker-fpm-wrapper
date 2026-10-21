<?php

// The request is longer than request_slowlog_timeout, so php-fpm writes the trace to the slowlog.

function slowOuter()
{
    slowInner();
}

function slowInner()
{
    sleep(2);
}

slowOuter();

echo 'ok';
