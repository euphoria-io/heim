import React from 'react'


export default React.createClass({
  propTypes: {
    href: React.PropTypes.string.isRequired,
    color: React.PropTypes.string,
    textDecoration: React.PropTypes.string,
    style: React.PropTypes.object,
    children: React.PropTypes.node,
  },

  getDefaultProps() {
    return {
      textDecoration: 'underline',
    }
  },

  render() {
    return (
      <a href={this.props.href} target="_blank" style={{
        color: this.props.color,
        textDecoration: this.props.textDecoration,
        ...this.props.style,
      }}>{this.props.children}</a>
    )
  },
})
