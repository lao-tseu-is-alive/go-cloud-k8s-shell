#!/bin/bash
trivy k8s --severity MEDIUM,HIGH,CRITICAL --report summary cluster
