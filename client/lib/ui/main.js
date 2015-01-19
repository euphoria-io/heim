var React = require('react/addons')
var Reflux = require('reflux')
var cx = React.addons.classSet

var actions = require('../actions')
var Scroller = require('./scroller')
var Messages = require('./messages')
var UserList = require('./userlist')
var NotifyToggle = require('./notifytoggle')


module.exports = React.createClass({
  displayName: 'Main',

  mixins: [
    Reflux.connect(require('../stores/chat').store, 'chat'),
    Reflux.connect(require('../stores/focus').store, 'focus'),
  ],

  onScrollbarSize: function(width) {
    this.setState({scrollbarWidth: width})
  },

  toggleSettings: function() {
    this.setState({settingsOpen: !this.state.settingsOpen})
  },

  onMouseDown: function() {
    // FIXME: preventing/canceling a mousedown in React doesn't seem to stop
    // the subsequent click event, so we have to resort to this hack.
    this._isFocusClick = Date.now() - this.state.focus.focusChangedAt < 50
  },

  onClick: function(ev) {
    // prevent clicks to focus window and link clicks from triggering elements
    if (this._isFocusClick || ev.target.nodeName == 'A') {
      actions.focusEntry()
      ev.stopPropagation()
      return
    }

    if (ev.target.nodeName == 'INPUT' || window.getSelection().type == 'Range') {
      return
    }

    actions.focusEntry()
  },

  render: function() {
    return (
      <div className="chat">
        <Scroller target=".entry" bottomSpace={75} className={cx({'messages-container': true, 'form-focus': this.state.focus.windowFocused && this.state.chat.connected})} onScrollbarSize={this.onScrollbarSize} onNearTop={actions.loadMoreLogs}>
          <div className="messages-content" onMouseDownCapture={this.onMouseDown} onClickCapture={this.onClick}>
            <div className="top-right" style={{marginRight: this.state.scrollbarWidth}}>
              <div className="settings-pane">
                {this.state.settingsOpen && <NotifyToggle />}
                <button type="button" className="settings" onClick={this.toggleSettings} />
              </div>
              <UserList users={this.state.chat.who} />
            </div>
            <Messages ref="messages" />
          </div>
        </Scroller>
      </div>
    )
  },
})
