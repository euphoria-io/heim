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
    <common.TopBubbleBox logo="logo-warning.png" padding={15}>
      <Item align="center">
        <Text {...textDefaults} fontSize={24}>would you like to reset your password?</Text>
      </Item>
    </common.TopBubbleBox>
    <common.BodyBox>
      <Item>
        <Text {...textDefaults}>hey {'{{AccountName}}'}, we've received a password reset request for your <Link {...textDefaults} href="{{SiteURL}}">{'{{SiteName}}'}</Link> account:</Text>
      </Item>
      <common.BigButton color="#dca955" href="{{ResetPasswordURL}}">
        reset your password
      </common.BigButton>
      <Item>
        <Text {...textDefaults}>if you did not make this request and suspect something fishy is going on, please reply to this email immediately.</Text>
      </Item>
    </common.BodyBox>
    <common.Footer>
      {common.standardFooter}
    </common.Footer>
  </common.StandardEmail>
)
