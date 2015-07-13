// ignore Text, Image redefinition
// jshint -W079

var React = require('react')

var email = require('./email')
var Email = email.Email
var Box = email.Box
var Item = email.Item
var Text = email.Text
var Link = email.Link
var Image = email.Image


var textDefaults = module.exports.textDefaults = {
  fontFamily: 'Verdana, sans-serif',
  fontSize: '16px',
  color: '#4d4d4d',
}

module.exports.StandardEmail = React.createClass({
  render: function() {
    return (
      <Email title="{{EmailSubject}}" bgcolor="#f0f0f0" cellSpacing="30">
        {this.props.children}
      </Email>
    )
  },
})

module.exports.TopBubbleBox = React.createClass({
  getDefaultProps: function() {
    return {
      padding: 7
    }
  },
  render: function() {
    return (
      <Item align="center">
        <Link href="{{SiteURL}}">
          <Image src={this.props.logo} width={67} height={90} />
        </Link>
        <Box width="600" cellPadding="2" bgcolor="white" style={{
          borderBottom: '3px solid #ccc',
          borderRadius: '10px',
          padding: this.props.padding,
        }}>
          {this.props.children}
        </Box>
      </Item>
    )
  },
})

module.exports.BodyBox = React.createClass({
  render: function() {
    return (
      <Item>
        <Box cellSpacing={20} width="100%" bgcolor="white" style={{borderBottom: '3px solid #ccc'}}>
          {this.props.children}
        </Box>
      </Item>
    )
  },
})

module.exports.BigButton = React.createClass({
  render: function() {
    return (
      <Item align="center" cellPadding={24}>
        <Link color="white" textDecoration="none" href={this.props.href} style={{
          display: 'inline-block',
          background: this.props.color,
          padding: '22px 30px',
          borderRadius: '4px',
        }}>
          <Text {...textDefaults} fontSize={24} fontWeight="bold" color="white">{this.props.children}</Text>
        </Link>
      </Item>
    )
  },
})

module.exports.Footer = React.createClass({
  render: function() {
    return (
      <Item style={{paddingLeft: '20px'}}>
        {this.props.children}
      </Item>
    )
  },
})

module.exports.standardFooter = (
  <Text {...textDefaults} fontSize={13} color="#7d7d7d">
    this message was sent to <Link {...textDefaults} textDecoration="none" href="mailto:{{AccountEmailAddress}}">{'{{AccountEmailAddress}}'}</Link> because an account is registered on <Link {...textDefaults} textDecoration="none" href="{{SiteURL}}">{'{{SiteURLShort}}'}</Link> with this email address.
    would you like to change your <Link {...textDefaults} href="{{AccountEmailPreferencesURL}}">email notification preferences</Link>?
  </Text>
)
