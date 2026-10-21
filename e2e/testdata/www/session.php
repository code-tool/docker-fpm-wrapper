<?php

declare(strict_types=1);

// Starts the session with the ID from the query. The request keeps the session file open
// for the number of seconds in "seconds".

@mkdir(session_save_path(), 0700, true);

session_id($_GET['id']);
session_start();
$_SESSION['time'] = time();

sleep((int) ($_GET['seconds'] ?? 0));
session_write_close();

echo 'ok';
