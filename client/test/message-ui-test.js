require('./support/setup')
var assert = require('assert')
var React = require('react/addons')
var TestUtils = React.addons.TestUtils
var Immutable = require('immutable')

describe('<Message>', function() {
  var Message = require('../lib/ui/message')

  it('automatically links urls', function() {
    var message = TestUtils.renderIntoDocument(
      <Message message={Immutable.fromJS({
        time: 123456,
        sender: {hue: 128},
        content: 'http://google.com',
      })} />
    )

    var messageContent = TestUtils.findRenderedDOMComponentWithClass(message, 'message')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com" target="_blank">google.com</a>')
  })

  it('truncates long urls', function() {
    var message = TestUtils.renderIntoDocument(
      <Message message={Immutable.fromJS({
        time: 123456,
        sender: {hue: 128},
        content: 'http://google.com/abcdefghijklmnopqrstuvwxyz1234567890',
      })} />
    )

    var messageContent = TestUtils.findRenderedDOMComponentWithClass(message, 'message')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com/abcdefghijklmnopqrstuvwxyz1234567890" target="_blank">google.com/abcdefghijklmnopqrstuvwxyz1..</a>')
  })
})
