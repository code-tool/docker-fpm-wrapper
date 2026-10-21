<?php

sleep(isset($_GET['seconds']) ? (int) $_GET['seconds'] : 1);

echo 'done';
