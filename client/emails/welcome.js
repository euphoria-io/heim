// ignore Text, Image redefinition
// jshint -W079

var React = require('react')

var email = require('./email')
var Item = email.Item
var Text = email.Text
var Link = email.Link
var common = require('./common')
var textDefaults = common.textDefaults


module.exports = (
  <common.StandardEmail>
    <common.TopBubbleBox logo="logo.png">
      <Item align="center">
        <Text {...textDefaults} fontSize={52}>hi!</Text>
      </Item>
      <Item align="center">
        <Text {...textDefaults} fontSize={18} color="#9f9f9f">welcome to {'{{.SiteName}}'} :)</Text>
      </Item>
    </common.TopBubbleBox>
    <common.BodyBox>
      <Item align="center">
        <Text {...textDefaults}>your account is almost ready:</Text>
      </Item>
      <common.BigButton color="#80c080" href="{{.VerifyEmailURL}}">
        verify your email address
      </common.BigButton>
      <Item>
        <Text {...textDefaults}>we hope you have a wonderful time on <Link {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</Link>. if you have any questions or comments, feel free to <Link {...textDefaults} href="mailto:{{.ContactEmailAddress}}">contact us</Link>.</Text>
      </Item>
    </common.BodyBox>
    <common.Footer>
      <Text {...textDefaults} fontSize={13} color="#7d7d7d">this message was sent to <Link {...textDefaults} textDecoration="none" href="mailto:{{.AccountEmailAddress}}">{'{{.AccountEmailAddress}}'}</Link> because someone signed up for an account on <Link {...textDefaults} textDecoration="none" href="{{.SiteURL}}">{'{{.SiteURLShort}}'}</Link> with this email address. if you did not request this email, please disregard.</Text>
    </common.Footer>
  </common.StandardEmail>
)
