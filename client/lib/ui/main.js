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
  ],

  componentDidMount: function() {
    window.addEventListener('blur', this.onWindowBlur, false)
  },

  componentWillUnmount: function() {
    window.removeEventListener('blur', this.onWindowBlur, false)
  },

  getInitialState: function() {
    return {formFocus: false, settingsOpen: false}
  },

  focusInput: function(ev) {
    if (ev.target.nodeName == 'INPUT' || window.getSelection().type == 'Range') {
      return
    }

    this.refs.messages.focusInput()
  },

  onScrollbarSize: function(width) {
    this.setState({scrollbarWidth: width})
  },

  onFormFocus: function() {
    this.setState({formFocus: true})
  },

  onMouseUp: function(ev) {
    if (!this.refs.messages.isFocused()) {
      this.setState({formFocus: false})
    }
  },

  onWindowBlur: function() {
    this.setState({formFocus: false})
  },

  toggleSettings: function() {
    this.setState({settingsOpen: !this.state.settingsOpen})
  },

  render: function() {
    return (
      <div className="chat" onMouseUp={this.onMouseUp}>
        <Scroller className={cx({'messages-container': true, 'settings-open': this.state.settingsOpen, 'form-focus': this.state.formFocus})} onClick={this.focusInput} onScrollbarSize={this.onScrollbarSize}>
          <div className="messages-content">
            <button type="button" className="settings" onClick={this.toggleSettings} />
            <div className="top-right" style={{marginRight: this.state.scrollbarWidth}}>
              <UserList users={this.state.chat.who} />
            </div>
            <Messages ref="messages" onFormFocus={this.onFormFocus} />
          </div>
          <div className="settings-pane">
            <NotifyToggle />
          </div>
        </Scroller>
      </div>
    )
  },
})
