#!/bin/bash

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"

set -e

readonly samples="${d}/samples"
readonly cronjob_sample="${samples}/v1_cronjob.yaml"
readonly job_sample="${samples}/v1_job.yaml"
readonly podprofile_sample="${samples}/v1_podprofile.yaml"
readonly results="${d}/results"
readonly cronjob_result="${results}/cronjob.yaml"
readonly job_result="${results}/job.yaml"

echo '## Examples'

echo '### Example of CronJob'
echo 'There are [PodProfile](#podprofile) and [CronJob](#cronjob):'
echo '``` yaml'
cat "${podprofile_sample}"
echo '---'
cat "${cronjob_sample}"
echo '```'
echo "These manifests will create the following resources:"
echo '``` yaml'
cat "${cronjob_result}"
echo '```'

echo '### Example of Job'
echo 'There are [PodProfile](#podprofile) and [Job](#job):'
echo '``` yaml'
cat "${podprofile_sample}"
echo '---'
cat "${job_sample}"
echo '```'
echo "These manifests will create the following resources:"
echo '``` yaml'
cat "${job_result}"
echo '```'
