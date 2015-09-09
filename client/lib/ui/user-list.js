var React = require('react')

var MessageText = require('./message-text')


module.exports = React.createClass({
  displayName: 'UserList',

  mixins: [require('react-immutable-render-mixin')],

  render: function() {
    var list

    list = this.props.users
      .toSeq()
      .filter(user => user.get('name'))

    list = list.sortBy(user => user.get('name').toLowerCase())

    return (
      <div className="user-list" {...this.props}>
        {list.map(function(user) {
          return <MessageText key={user.get('session_id')} className="nick" onlyEmoji={true} style={{background: 'hsl(' + user.get('hue') + ', 65%, 85%)'}} content={user.get('name')} />
        }, this).toArray()}
      </div>
    )
  },
})
