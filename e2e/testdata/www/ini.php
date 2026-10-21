<?php

declare(strict_types=1);

echo ini_get((string) ($_GET['name'] ?? ''));
