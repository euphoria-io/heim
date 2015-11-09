import React from 'react'
import classNames from 'classnames'


export default React.createClass({
  displayName: 'CheckField',

  propTypes: {
    name: React.PropTypes.string.isRequired,
    value: React.PropTypes.bool,
    onModify: React.PropTypes.func,
    onValidate: React.PropTypes.func,
    className: React.PropTypes.string,
    tabIndex: React.PropTypes.number,
    disabled: React.PropTypes.bool,
    children: React.PropTypes.node,
  },

  onChange(ev) {
    this.props.onModify(ev.target.checked)
  },

  focus() {
    this.refs.input.focus()
  },

  render() {
    return (
      <div className={classNames('check-field', this.props.className)}>
        <input
          ref="input"
          type="checkbox"
          tabIndex={this.props.tabIndex}
          name={this.props.name}
          id={'field-' + this.props.name}
          disabled={this.props.disabled}
          checked={this.props.value}
          onChange={this.onChange}
        />
        <label htmlFor={'field-' + this.props.name}>{this.props.children}</label>
      </div>
    )
  },
})
