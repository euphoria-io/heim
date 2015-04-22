var React = require('react/addons')
var Reflux = require('reflux')
var classNames = require('classnames')
var moment = require('moment')


module.exports = React.createClass({
  displayName: 'LiveTimeAgo',

  mixins: [
    Reflux.connect(require('../stores/clock').minute, 'now'),
  ],

  render: function() {
    var t = this.props.time
    if (!moment.isMoment(t)) {
      t = moment.unix(t)
    }

    var display
    var className
    if (moment(this.state.now).diff(t, 'minutes') === 0) {
      display = this.props.nowText
      className = 'now'
    } else {
      display = t.locale('en-short').from(this.state.now, true)
    }

    return (
      <time dateTime={t.toISOString()} title={t.format('MMMM Do YYYY, h:mm:ss a')} {...this.props} className={classNames(className, this.props.className)}>
        {display}
      </time>
    )
  },
})
