#!/bin/sh

ssh -o StrictHostKeychecking=no -i keys/vpn ubuntu@$(terraform output --raw ip)
