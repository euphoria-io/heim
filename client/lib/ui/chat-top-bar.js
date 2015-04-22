var React = require('react/addons')
var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup

var update = require('../stores/update')
var FastButton = require('./fast-button')
var UserList = require('./user-list')
var ToggleBubble = require('./toggle-bubble')
var RoomTitle = require('./room-title')


module.exports = React.createClass({
  displayName: 'ChatTopBar',

  mixins: [require('react-immutable-render-mixin')],

  showUserList: function() {
    this.refs.userList.toggle()
  },

  showUpdateBubble: function() {
    this.refs.updateBubble.toggle()
  },

  render: function() {
    var userCount = this.props.who.filter(user => user.get('name')).size

    // use an outer container element so we can z-index the bar above the
    // bubbles. this makes the bubbles slide from "underneath" the bar.
    return (
      <div className="top-bar">
        <RoomTitle name={this.props.roomName} authType={this.props.authType} connected={this.props.connected} joined={this.props.joined} />
        <div className="right">
          <ReactCSSTransitionGroup transitionName="spinner">{this.props.working && <div key="spinner" className="spinner" />}</ReactCSSTransitionGroup>
          {this.props.updateReady && <FastButton fastTouch className="update-available" onClick={this.showUpdateBubble} />}
          {this.props.joined && <FastButton fastTouch className="user-count" onClick={this.showUserList}>{userCount}</FastButton>}
        </div>
        <ToggleBubble ref="userList" className="users" sticky={true}>
          {userCount > 0 ? <UserList users={this.props.who} /> : <div className="nick">nobody here :(</div>}
        </ToggleBubble>
        <ToggleBubble ref="updateBubble" className="update">
          <FastButton className="update-button" onClick={update.perform}><p>update ready<em>{Heim.isTouch ? 'tap' : 'click'} to reload</em></p></FastButton>
        </ToggleBubble>
      </div>
    )
  },
})
