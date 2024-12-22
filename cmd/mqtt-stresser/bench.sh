#!/bin/bash

./mqtt-stresser-"$(uname)"-"$(arch)" -broker tcp://localhost:1883 -num-clients=2 -num-messages=10000


