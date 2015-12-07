import React from 'react'
import Reflux from 'reflux'
import classNames from 'classnames'
import moment from 'moment'


function checkIsMoment(props, propName) {
  if (!moment.isMoment(props[propName])) {
    return new Error('not a Moment instance')
  }
}

export default React.createClass({
  displayName: 'LiveTimeAgo',

  propTypes: {
    time: React.PropTypes.oneOfType([React.PropTypes.number, checkIsMoment]),
    nowText: React.PropTypes.string,
    className: React.PropTypes.string,
  },

  mixins: [
    Reflux.connect(require('../stores/clock').minute, 'now'),
  ],

  render() {
    let t = this.props.time
    if (!moment.isMoment(t)) {
      t = moment.unix(t)
    }

    let display
    let className
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
