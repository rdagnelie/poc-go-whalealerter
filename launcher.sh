#!/bin/bash
sudo docker build -t whalealerter .
sudo docker run --env-file .env -it --rm --name whalealerter whalealerter
