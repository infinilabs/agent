SHELL=/bin/bash

# APP info
APP_NAME := agent
APP_VERSION := 1.0.0_SNAPSHOT
APP_CONFIG := $(APP_NAME).yml
APP_EOLDate ?= "2026-12-31T10:10:10Z"
APP_STATIC_FOLDER := .public
APP_STATIC_PACKAGE := public
APP_UI_FOLDER := ui
APP_PLUGIN_FOLDER := plugin
FRAMEWORK_BRANCH := master
FRAMEWORK_VENDOR_BRANCH := master

include ../framework/Makefile
