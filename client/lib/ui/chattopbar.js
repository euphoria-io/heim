var React = require('react/addons')

var update = require('../stores/update')
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

  showUpdateBubble: function() {
    this.refs.updateBubble.toggle()
  },

  render: function() {
    var userCount = this.props.who.filter(user => user.get('name')).size

    // use an outer container element so we can z-index the bar above the
    // bubbles. this makes the bubbles slide from "underneath" the bar.
    return (
      <div className="topbar-container" style={{marginRight: this.props.scrollbarWidth + 1}}>
        <div className="topbar">
          <RoomTitle name={this.props.roomName} authType={this.props.authType} />
          <div className="right">
            {this.props.updateReady && <FastButton fastTouch className="update" onClick={this.showUpdateBubble} />}
            <FastButton fastTouch className="user-count" onClick={this.showUserList}>{userCount}</FastButton>
          </div>
        </div>
        <Bubble ref="userList" className="users" rightOffset={this.props.scrollbarWidth + 1}>
          {userCount > 0 ? <UserList users={this.props.who} /> : <div className="nick">nobody here</div>}
        </Bubble>
        <Bubble ref="updateBubble" className="update" rightOffset={this.props.scrollbarWidth + 1}>
          <FastButton className="update-button" onClick={update.perform}><p>update ready<em>{Heim.isTouch ? 'tap' : 'click'} to reload</em></p></FastButton>
        </Bubble>
      </div>
    )
  },
})
