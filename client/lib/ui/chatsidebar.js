var React = require('react/addons')
var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup
var Reflux = require('reflux')

var actions = require('../actions')
var UserList = require('./userlist')
var NotifyToggle = require('./notifytoggle')
var PrivacyBubble = require('./privacybubble')


module.exports = React.createClass({
  displayName: 'ChatSidebar',

  mixins: [
    require('react-immutable-render-mixin'),
    Reflux.listenTo(actions.showSettings, 'showSettings'),
  ],

  getInitialState: function() {
    return {
      settingsOpen: false,
      userListCollapsed: true,
    }
  },

  toggleSettings: function() {
    this.setState({settingsOpen: !this.state.settingsOpen})
  },

  showSettings: function() {
    this.setState({settingsOpen: true})
  },

  expandUserList: function() {
    this.setState({userListCollapsed: false})
  },

  collapseUserList: function() {
    this.setState({userListCollapsed: true})
  },

  showPrivacyInfo: function() {
    this.refs.privacyInfo.show()
  },

  render: function() {
    return (
      <div className="sidebar" style={{marginRight: this.props.scrollbarWidth}}>
        <div className="top-line">
          <ReactCSSTransitionGroup transitionName="settings">
            {this.state.settingsOpen &&
              <span key="content" className="settings-content">
                <NotifyToggle />
              </span>
            }
          </ReactCSSTransitionGroup>
          <span className="room">
            <a className="name" href={'/room/' + this.props.roomName} onClick={ev => ev.preventDefault()}>&amp;{this.props.roomName}</a>
            {this.props.authType && <button className="private" onClick={this.showPrivacyInfo}>private</button>}
          </span>
          <button type="button" className="settings" onClick={this.toggleSettings} tabIndex="-1" />
        </div>
        <UserList users={this.props.who} collapsed={this.state.userListCollapsed} onMouseEnter={this.expandUserList} onMouseLeave={this.collapseUserList} />
        <PrivacyBubble ref="privacyInfo" authType={this.props.authType} rightOffset={this.props.scrollbarWidth} />
        {this.props.roomName == 'space' && <div className="norman"><p>norman</p><img src="//i.imgur.com/wAz2oho.jpg" /></div>}
      </div>
    )
  },
})
