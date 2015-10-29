import React from 'react'


export default React.createClass({
  propTypes: {
    fontFamily: React.PropTypes.string,
    fontSize: React.PropTypes.number,
    fontWeight: React.PropTypes.string,
    lineHeight: React.PropTypes.number,
    color: React.PropTypes.string,
    style: React.PropTypes.object,
    children: React.PropTypes.node,
  },

  getDefaultProps() {
    return {
      fontFamily: 'sans-serif',
      fontSize: 14,
      color: '#000',
    }
  },

  render() {
    return (
      <span style={{
        fontFamily: this.props.fontFamily,
        fontSize: this.props.fontSize,
        fontWeight: this.props.fontWeight,
        lineHeight: this.props.lineHeight !== null ? this.props.lineHeight : this.props.fontSize,
        color: this.props.color,
        ...this.props.style,
      }}>{this.props.children}</span>
    )
  },
})

