var React = require('react/addons')

var FastButton = require('./fastbutton')
var UserList = require('./userlist')
var Bubble = require('./bubble')
var RoomTitle = require('./roomtitle')


module.exports = React.createClass({
  displayName: 'ChatTopBar',

  mixins: [require('react-immutable-render-mixin')],

  showUserList: function() {
    this.refs.userList.toggle()
  },

  render: function() {
    var userCount = this.props.who.filter(user => user.get('name')).size

    // use an outer container element so we can z-index the bar above the
    // bubbles. this makes the bubbles slide from "underneath" the bar.
    return (
      <div className="topbar-container" style={{marginRight: this.props.scrollbarWidth + 1}}>
        <div className="topbar">
          <RoomTitle name={this.props.roomName} authType={this.props.authType} />
          {userCount > 0 && <FastButton fastTouch className="user-count" onClick={this.showUserList}>{userCount}</FastButton>}
        </div>
        <Bubble ref="userList" className="users" rightOffset={this.props.scrollbarWidth + 1}>
          <UserList users={this.props.who} />
        </Bubble>
      </div>
    )
  },
})
