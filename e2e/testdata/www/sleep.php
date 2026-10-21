<?php

declare(strict_types=1);

sleep((int) ($_GET['seconds'] ?? 1));

echo 'done';
