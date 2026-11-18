#!/bin/bash

go build -o batocera-scrap-manager . && ./batocera-scrap-manager serve --addr 0.0.0.0:8087
