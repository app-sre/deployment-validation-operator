#!/bin/bash

make docker-login-and-push
BRANCH_CHANNEL=staging make channel-catalog
BRANCH_CHANNEL=production REMOVE_UNDEPLOYED=true make channel-catalog
