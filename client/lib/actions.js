var Reflux = require('reflux')

module.exports = Reflux.createActions([
  'sendMessage',
  'focusMessage',
  'toggleFocusMessage',
  'setEntryText',
  'focusEntry',
  'loadMoreLogs',
  'setNick',
  'connect',
])
