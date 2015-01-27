var Reflux = require('reflux')

module.exports = Reflux.createActions([
  'sendMessage',
  'focusMessage',
  'toggleFocusMessage',
  'setEntryText',
  'focusEntry',
  'scrollToEntry',
  'keydownOnEntry',
  'loadMoreLogs',
  'showSettings',
  'setNick',
  'connect',
])

module.exports.connect.sync = true
module.exports.keydownOnEntry.sync = true
