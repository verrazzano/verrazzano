#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# The absolute file and directory path of the calling script.
_LOGGING_CALLER_NAME="${BASH_SOURCE[${#BASH_SOURCE[@]} - 1]}"
_LOGGING_CALLER_PATH="$(cd $(dirname ${_LOGGING_CALLER_NAME}); pwd -P)/$(basename ${_LOGGING_CALLER_NAME})"
_LOGGING_CALLER_DIR="$(cd $(dirname ${_LOGGING_CALLER_PATH}); pwd -P)"

# Determine the log file location.
export LOG_FILE="${LOG_FILE:-${_LOGGING_CALLER_DIR}/build/logs/$(basename ${_LOGGING_CALLER_NAME}).log}"
export LOGFILE="${LOG_FILE}"
# Ensure log dir exists.
if [ ! -d "$(dirname ${LOG_FILE})" ] ; then
  mkdir -p "$(dirname ${LOG_FILE})"
fi

# Duplicate stdout and stderr file descriptors.
# Only do this if they have not already been duplicated.
if [ -z "${_LOGGING_CONSOLE_STDOUT:-}" ]; then
  export _LOGGING_CONSOLE_STDOUT=4 #"/dev/fd/5"
  export _LOGGING_CONSOLE_STDERR=5 #"/dev/fd/6"
  exec 4>&1 5<&2

  # Redirect stdout and stderr for this shell to a log file.
  echo "Output redirected to ${LOG_FILE}"
  if [ ${DEBUG:-0} -ge 4 ]; then
    exec 1>> "$LOG_FILE" 2>&1

    # Enable full shell trace logging.
    set -x
  else
    # The >(...) portion prefixes a timestamp to every line of the merged stdout/stderr.
    exec &> >( while IFS= read -r l; do printf '[%s] %s\n' "$(date -u '+%Y-%m-%d %H:%M:%S %Z')" "$l"; done 1>> ${LOG_FILE} )
  fi
  echo "Output captured for ${_LOGGING_CALLER_PATH}"
fi

###################################################################################################
# The functions below deals writing output to both the console and the log file.
###################################################################################################

_LOGGING_ACTION_BEGLINE_CODES='\033[0G'
_LOGGING_ACTION_SUCCESS_CODES='\033[1;32m'
_LOGGING_ACTION_FAILURE_CODES='\033[1;31m'
_LOGGING_ACTION_SECTION_CODES='\033[1m'
_LOGGING_ACTION_NORMAL_CODES='\033[0;39m'
_LOGGING_ACTION_FORMAT='%-77s[%6s]'

# Determine if the action spinner should be enabled.
# The spinner will not be enabled when:
#   - DISABLE_SPINNER is set, or
#   - standard input is not a terminal, or
#   - DEBUG is set to a value greater than 0
# Internal function
function _logging_action_spinner_enabled() {
  if [ -t 0 ] && [ -z "${DISABLE_SPINNER:-}" ] && [ ${DEBUG:-0} -le 0 ]; then
    return 0
  else
    return 1
  fi
}

# Determine if the color control codes should be enabled.
# The control codes will not be enabled when any of the below are true:
#   - DISABLE_SPINNER is set
#   - standard input is not a terminal (i.e. ![ -t 0 ])
# Internal function
function _logging_action_color_enabled() {
  if [ -t 0 ] && [ -z "${DISABLE_SPINNER:-}" ]; then
    return 0
  else
    return 1
  fi
}

# Write action sart message to log file and display the action to the console.
# Internal function
function _logging_action_started() {
  local msg="${_logging_action_msg:-}"
  local status=" .... "
  local endlin="\n"
  # Remember the current shell pid to prevent sub scripts from killing spinner.
  export _logging_action_pid=$$
  # Remember the logging action message for use in success/failure/cleanup functions.
  export _logging_action_msg="${msg}"

  printf "${_LOGGING_ACTION_FORMAT}\n" "${msg}" "${status}"
  if _logging_action_spinner_enabled; then
    status='      '
    endlin=''
  fi
  printf "${_LOGGING_ACTION_FORMAT}${endlin}" "${msg}" "${status}" >&${_LOGGING_CONSOLE_STDOUT}

  if _logging_action_spinner_enabled; then
    # Execute the spinner function is an asynchronous sub-shell.
    _logging_action_spinner &
    export _logging_action_spinner_pid=$!
    # Disown spinner pid to prevent bash's job control from emitting kill message.
    disown ${_logging_action_spinner_pid}
  fi
}

# Write action success message to log file and update console action status with success.
# Internal function
function _logging_action_success() {
  local msg="${_logging_action_msg:-}"
  local status='  OK  '
  local beglin=''
  local format="${_LOGGING_ACTION_FORMAT}"

  if _logging_action_spinner_enabled; then
    beglin="${_LOGGING_ACTION_BEGLINE_CODES}"
  fi
  if _logging_action_color_enabled; then
    format="${beglin}%-77s[${_LOGGING_ACTION_SUCCESS_CODES}%6s${_LOGGING_ACTION_NORMAL_CODES}]"
  fi

  printf "${_LOGGING_ACTION_FORMAT}\n" "${msg}" "${status}"
  printf "${format}\n" "${msg}" "${status}" >&${_LOGGING_CONSOLE_STDOUT}
}

# Write action failure message to log file and update console action status with failure.
# Internal function
function _logging_action_failure() {
  local msg="${_logging_action_msg:-}"
  local status='FAILED'
  local beglin=''
  local format="${_LOGGING_ACTION_FORMAT}"

  if _logging_action_spinner_enabled; then
    beglin="${_LOGGING_ACTION_BEGLINE_CODES}"
  fi
  if _logging_action_color_enabled; then
    format="${beglin}%-77s[${_LOGGING_ACTION_FAILURE_CODES}%6s${_LOGGING_ACTION_NORMAL_CODES}]"
  fi

  printf "${_LOGGING_ACTION_FORMAT}\n" "${msg}" "${status}"
  printf "${format}\n" "${msg}" "${status}" >&${_LOGGING_CONSOLE_STDOUT}
}

# Update console action status for a completsed action.
# Internal function
function _logging_action_complete() {
  # If this shell is the same one that started the action, then update the action status.
  if [ "${_logging_action_pid:-}" = "$$" ]; then
    if _logging_action_spinner_enabled; then
      _logging_action_spinner_kill
    fi
    local rc="${_logging_action_rc:-}"
    if [ "${rc}" = "0" ]; then
      _logging_action_success
    elif [ -n "${rc}" ]; then
      _logging_action_failure
    fi
    export _logging_action_rc=""
    export _logging_action_pid=""
  fi
  # If there is a failure message set output the message.
  if [ -n "${_logging_failure_msg:-}" ]; then
    printf "\n${_logging_failure_msg}\n"
    printf "\n${_logging_failure_msg}\n" >&${_LOGGING_CONSOLE_STDERR}
    export _logging_failure_msg=""
  fi
}

# Loop until killed updating the console action status with a spinner.
# Will only be called if stdin is open on a terminal.
# Internal function
function _logging_action_spinner() {
  local msg="${_logging_action_msg:-}"
  local spinner_chars='\|/-'
  local status=''
  while :
  do
    for i in `seq 0 3`
    do
      status="  ${spinner_chars:$i:1}   "
      printf "${_LOGGING_ACTION_BEGLINE_CODES}${_LOGGING_ACTION_FORMAT}" "${msg}" "${status}" >&${_LOGGING_CONSOLE_STDOUT}
      sleep .1
    done
  done
}

# Kill the spinner process if the pod it set.
# Internal function
function _logging_action_spinner_kill() {
  local pid="${_logging_action_spinner_pid:-}"
  if [ -n "${pid}" ]; then
    if kill -0 ${pid} 2>/dev/null ; then
      kill -9 ${pid} 2>/dev/null || true
      wait ${pid} 2>/dev/null || true
      export _logging_action_spinner_pid=""
    fi
  fi
}

# Execute a command with a status output.
# The return code of the command is used to populate the final status field.
# If a terminal is used, the status will be updated with a "spinner" while the command is executing.
# $1: The message to be written to the console's stdout and the log file.
# $@: The command or function to execute.  Executed as `eval "$@"`.
# Returns the result of the executed `eval "$@"` command.
function action() {
  local msg="$1"
  shift
  local cmd="$@"
  local rc

  export _logging_action_msg="${msg}"
  _logging_action_started
  export _logging_action_rc="?"
  eval $cmd && rc=0 || rc=$?
  export _logging_action_rc="${rc}"
  _logging_action_complete

  return $rc
}

# Log a section (i.e. bold) message to the console's standard output and the log file.
# This should not be invoked within an action command as this disrupts console output formatting.
# $@: The message to be written to the console's stdout and the log file.  Written as "${@}".
# Returns 0
function section() {
  local msg="$@"
  echo "${msg}"
  if [ -n "${_logging_action_rc:-}" ] && _logging_action_spinner_enabled; then
    echo -en "\n" >&${_LOGGING_CONSOLE_STDOUT}
  fi
  if _logging_action_color_enabled; then
    printf "${_LOGGING_ACTION_SECTION_CODES}%s${_LOGGING_ACTION_NORMAL_CODES}\n" "${msg}" >&${_LOGGING_CONSOLE_STDOUT}
  else
    printf "${msg}\n" >&${_LOGGING_CONSOLE_STDOUT}
  fi
}

# Log a message to the console's standard output and the log file.
# This should not be invoked within an action command as this disrupts console output formatting.
# $@: The message to be written to the console's stdout and the log file.  Written as "${@}".
# Returns 0
function status() {
  local msg="$@"
  echo "${msg}"
  if [ -n "${_logging_action_rc:-}" ] && _logging_action_spinner_enabled; then
    echo -e "\n${msg}" >&${_LOGGING_CONSOLE_STDOUT}
  else
    echo "${msg}" >&${_LOGGING_CONSOLE_STDOUT}
  fi
}

# Log a message to the console's standard error and the log file.
# This should not be invoked within an action command as this disrupts console output formatting.
# The console's standard output and standard error are not synchronized.
# Messages written to standard error may appear out of order from those written to standard output.
# $@: The message to be written to the console's standard error and the log file.  Written as "${@}".
# Returns 0
function error() {
  local msg="$@"
  echo "${msg}"
  if [ -n "${_logging_action_rc:-}" ] && _logging_action_spinner_enabled; then
    echo -e "\n${msg}" >&${_LOGGING_CONSOLE_STDERR}
  else
    echo "${msg}" >&${_LOGGING_CONSOLE_STDERR}
  fi
}

# Log a message to the console's standard error and the log file and exit with a status code.
# May be invoke with no parameters. When invoked with single parameter that parameter must be a message.
# Default exit code is 1.
# $1: The message to be written to the console's standard output and the log file.  Written as "$1".  Defaults to "".
# $2: The exit code.  Defaults to 1.
# Exits with the $2 exit code.
function fail() {
  local msg="${1:-}"
  local rc=${2:-1}
  export _logging_failure_msg="${msg}"
  exit ${rc}
}

###################################################################################################
# The functions below deal with writing output to the log file.
###################################################################################################

# Write a message to the log file if DEBUG >= 0.
# Also write a message to the console's stdout if DEBUG >=1.
# $@: The message to write.  Written as "${@}".
# Returns 0
function log() {
  local msg="$@"
  if [ ${DEBUG:-0} -ge 0 ]; then
    echo -e "${msg}"
  fi
  if [ ${DEBUG:-0} -ge 1 ]; then
    echo -e "${msg}" >&${_LOGGING_CONSOLE_STDOUT}
  fi
}

# Write a message to the log file and the console's stdout if DEBUG >= 2.
# $@: The message to write.  Written as "${@}".
# Returns 0
function debug() {
  local msg="$@"
  if [ ${DEBUG:-0} -ge 2 ]; then
    echo -e "${msg}" >&${_LOGGING_CONSOLE_STDOUT}
    echo -e "${msg}"
  fi
}

# Write a message to the log file and the console's stdout if DEBUG >= 3.
# $@: The message to write.  Written as "${@}".
# Returns 0
function trace() {
  local msg="$@"
  if [ ${DEBUG:-0} -ge 3 ]; then
    echo -e "${msg}" >&${_LOGGING_CONSOLE_STDOUT}
    echo -e "${msg}"
  fi
}

###################################################################################################
# The functions below deal with trapping signals to cleanup and complete logging.
###################################################################################################

# Generate a timestamp for logs.
# Internal function
function _logging_timestamp() {
  date -u '+%Y-%m-%d %H:%M%:%S %Z'
}

# Function used for interrupt (ie ^C) trap.
# Internal function
function _logging_interrupt_handler() {
  local rc=$1
  _logging_action_spinner_kill
  # If this is a real interrupt and not a call from the exit handler then log a message.
  if [ $rc -ge 128 ]; then
    # Pause briefly to help ensure stdout and stderr don't overlap.
    sleep 0
    msg="An interrupt occurred. Exiting with code $rc."
    if [ -n "${rc}" ] && _logging_action_spinner_enabled; then
      echo "" >&${_LOGGING_CONSOLE_STDOUT}
    fi
    echo -e "\n$msg See ${LOG_FILE} for details." >&${_LOGGING_CONSOLE_STDERR}
    echo -e "\n${msg}"
    # Remove the EXIT trap to avoid double logging of the exit message.
    trap - INT EXIT
    exit $rc
  fi
}

# Function used for EXIT (ie ^C) trap.
# Internal function
function _logging_exit_handler() {
  local rc=$1
  _logging_action_complete
  # If the exit code indicates a script error then output a message.
  # Return codes >= 128 usually indicate system exits (e.g. ^C)
  if [ $rc -gt 0 ] && [ $rc -lt 128 ]; then
    # Pause briefly to help ensure stdout and stderr don't overlap.
    sleep 0
    msg="A failure occurred. Exiting with code $rc."
    echo "" >&${_LOGGING_CONSOLE_STDERR}
    echo "${msg} See ${LOG_FILE} for details." >&${_LOGGING_CONSOLE_STDERR}
    echo ""
    echo "${msg}"
    # Remove the EXIT trap to avoid double logging of the exit message.
    trap - EXIT
  fi
  exit $rc
}

# Add handlers for interrupt and exit signals.
trap '_logging_interrupt_handler $?' INT
trap '_logging_exit_handler $?' EXIT