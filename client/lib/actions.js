var Reflux = require('reflux')

module.exports = Reflux.createActions([
  'sendMessage',
  'focusMessage',
  'toggleFocusMessage',
  'setEntryText',
  'focusEntry',
  'loadMoreLogs',
  'showSettings',
  'setNick',
  'connect',
])

module.exports.connect.sync = true
