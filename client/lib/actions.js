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

// sync so that we connect in the load tick
module.exports.connect.sync = true

// sync to allow entry to preventDefault keydown events
module.exports.keydownOnEntry.sync = true
