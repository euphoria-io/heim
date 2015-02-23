var React = require('react/addons')

var UserList = require('./userlist')
var Bubble = require('./bubble')
var PrivacyBubble = require('./privacybubble')


module.exports = React.createClass({
  displayName: 'ChatTopBar',

  mixins: [require('react-immutable-render-mixin')],

  showUserList: function() {
    this.refs.userList.show()
  },

  showPrivacyInfo: function() {
    this.refs.privacyInfo.show()
  },

  render: function() {
    var userCount = this.props.who.filter(user => user.get('name')).size

    // use an outer container element so we can z-index the bar above the
    // bubbles. this makes the bubbles slide from "underneath" the bar.
    return (
      <div className="topbar-container" style={{marginRight: this.props.scrollbarWidth + 1}}>
        <div className="topbar">
          <span className="room">
            <a className="name" href={'/room/' + this.props.roomName} onClick={ev => ev.preventDefault()}>&amp;{this.props.roomName}</a>
            {this.props.authType && <button className="private" onClick={this.showPrivacyInfo}>private</button>}
          </span>
          {userCount > 0 && <button className="nick user-count" onClick={this.showUserList} onTouchStart={this.showUserList}>{userCount}</button>}
        </div>
        <Bubble ref="userList" className="users" rightOffset={this.props.scrollbarWidth + 1}>
          <UserList users={this.props.who} />
        </Bubble>
        <PrivacyBubble ref="privacyInfo" authType={this.props.authType} />
      </div>
    )
  },
})
