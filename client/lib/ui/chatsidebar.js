var React = require('react/addons')
var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup
var Reflux = require('reflux')

var actions = require('../actions')
var update = require('../stores/update')
var UserList = require('./userlist')
var NotifyToggle = require('./notifytoggle')
var FastButton = require('./fastbutton')
var RoomTitle = require('./roomtitle')


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
          <RoomTitle name={this.props.roomName} authType={this.props.authType} rightOffset={this.props.scrollbarWidth} />
          <button type="button" className="settings" onClick={this.toggleSettings} tabIndex="-1" />
        </div>
        <UserList users={this.props.who} collapsed={this.state.userListCollapsed} onMouseEnter={this.expandUserList} onMouseLeave={this.collapseUserList} />
        {this.props.updateReady && <FastButton className="update-button" onClick={update.perform}><p>update ready<em>{Heim.isTouch ? 'tap' : 'click'} to reload</em></p></FastButton>}
        {this.props.roomName == 'space' && <div className="norman"><p>norman</p><img src="//i.imgur.com/45wJkX7.jpg" /></div>}
      </div>
    )
  },
})
