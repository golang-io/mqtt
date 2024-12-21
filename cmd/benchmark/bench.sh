#!/bin/bash

./mqtt-stresser -broker tcp://localhost:1883 -num-clients=2 -num-messages=10000 > 2_10000.log

