require('./support/setup')
var _ = require('lodash')
var assert = require('chai').assert
var Immutable = require('immutable')

var ChatTree = require('../lib/chat-tree')

describe('ChatTree', function() {
  describe('a new empty chat tree', function() {
    var tree = new ChatTree()

    it('should have an empty threads tree', function() {
      assert.equal(tree.threads.size, 0)
    })
  })

  var firstCount = Immutable.Map({
    descendants: 10,
    newDescendants: 1,
    ownDescendants: 1,
    mentionDescendants: 1,
    newMentionDescendants: 1,
    latestDescendantTime: 123,
    latestDescendant: 'abc',
  })

  var secondCount = Immutable.Map({
    descendants: 5,
    newDescendants: 1,
    ownDescendants: 1,
    mentionDescendants: 1,
    newMentionDescendants: 1,
    latestDescendantTime: 789,
    latestDescendant: 'xyz',
  })

  describe('merge count operation', function() {
    it('should add numeric fields together', function() {
      var mergedCount = ChatTree.mergeCount(firstCount, secondCount)
      assert.equal(mergedCount.get('descendants'), 15)
      assert.equal(mergedCount.get('newDescendants'), 2)
      assert.equal(mergedCount.get('ownDescendants'), 2)
      assert.equal(mergedCount.get('mentionDescendants'), 2)
      assert.equal(mergedCount.get('newMentionDescendants'), 2)
    })

    it('should choose the latest descendant time and id', function() {
      var mergedCount1 = ChatTree.mergeCount(firstCount, secondCount)
      var mergedCount2 = ChatTree.mergeCount(secondCount, firstCount)
      assert.equal(mergedCount1.get('latestDescendantTime'), 789)
      assert.equal(mergedCount2.get('latestDescendantTime'), 789)
      assert.equal(mergedCount1.get('latestDescendant'), 'xyz')
      assert.equal(mergedCount2.get('latestDescendant'), 'xyz')
    })
  })

  describe('subtract count operation', function() {
    it('should difference numeric fields together', function() {
      var mergedCount = ChatTree.subtractCount(firstCount, secondCount)
      assert.equal(mergedCount.get('descendants'), 5)
      assert.equal(mergedCount.get('newDescendants'), 0)
      assert.equal(mergedCount.get('ownDescendants'), 0)
      assert.equal(mergedCount.get('mentionDescendants'), 0)
      assert.equal(mergedCount.get('newMentionDescendants'), 0)
    })
  })

  var testMessages = [
    {
      '_seen': 1,
      '_own': true,
      'id':'message1',
      'parent': '__root',
      'time': 1,
      'content': 'hello!',
    },
    {
      '_seen': false,
      '_own': false,
      '_mention': true,
      'id':'message1-1',
      'parent': 'message1',
      'time': 11,
      'content': 'hey @test!',
    },
    {
      '_seen': false,
      '_own': true,
      '_mention': false,
      'id':'message1-1-1',
      'parent': 'message1-1',
      'time': 111,
      'content': 'long time no see!',
    },
    {
      '_seen': 2,
      '_own': false,
      'id':'message2',
      'parent': '__root',
      'time': 2,
      'content': 'hi!',
    },
    {
      '_seen': 21,
      '_own': false,
      'id':'message2-1',
      'parent': 'message2',
      'time': 21,
      'content': 'ayyy',
    },
    {
      '_seen': true,
      '_own': true,
      'id':'message2-2',
      'parent': 'message2',
      'time': 22,
      'content': 'sup',
    },
  ]

  function checkMessage1Counts(tree) {
    assert.deepEqual(tree.getCount('message1').toJS(), {
      descendants: 2,
      newDescendants: 1,
      ownDescendants: 1,
      mentionDescendants: 1,
      newMentionDescendants: 1,
      latestDescendantTime: 111,
      latestDescendant: 'message1-1-1',
    })

    assert.deepEqual(tree.getCount('message1-1').toJS(), {
      descendants: 1,
      newDescendants: 0,
      ownDescendants: 1,
      mentionDescendants: 0,
      newMentionDescendants: 0,
      latestDescendantTime: 111,
      latestDescendant: 'message1-1-1',
    })

    assert.deepEqual(tree.getCount('message1-1-1').toJS(), ChatTree.initCount.toJS())
  }

  var expectedMessage2Count = {
    descendants: 2,
    newDescendants: 0,
    ownDescendants: 1,
    mentionDescendants: 0,
    newMentionDescendants: 0,
    latestDescendantTime: 22,
    latestDescendant: 'message2-2',
  }

  function checkMessage2Counts(tree) {
    assert.deepEqual(tree.getCount('message2').toJS(), expectedMessage2Count)
    assert.deepEqual(tree.getCount('message2-1').toJS(), ChatTree.initCount.toJS())
    assert.deepEqual(tree.getCount('message2-2').toJS(), ChatTree.initCount.toJS())
  }

  describe('when adding a chain of nodes', function() {
    var tree

    beforeEach(function() {
      tree = new ChatTree()
    })

    it('should calculate correct counts', function() {
      tree.add(testMessages[0])
      assert.deepEqual(tree.getCount('message1').toJS(), ChatTree.initCount.toJS())

      tree.add(testMessages[1])
      assert.deepEqual(tree.getCount('message1').toJS(), {
        descendants: 1,
        newDescendants: 1,
        ownDescendants: 0,
        mentionDescendants: 1,
        newMentionDescendants: 1,
        latestDescendantTime: 11,
        latestDescendant: 'message1-1',
      })
      assert.deepEqual(tree.getCount('message1-1').toJS(), ChatTree.initCount.toJS())

      tree.add(testMessages[2])
      checkMessage1Counts(tree)
    })
  })

  describe('after adding a bunch of messages out of order', function() {
    var tree

    beforeEach(function() {
      tree = new ChatTree()
      tree.add(Immutable.Seq(testMessages).reverse().toArray())
    })

    it('should have the correct size', function() {
      assert.equal(tree.size, testMessages.length)
    })

    it('should calculate correct counts', function() {
      checkMessage1Counts(tree)
      checkMessage2Counts(tree)
    })

    it('should identify and score the correct threads', function() {
      assert.equal(tree.threads.size, 1)

      var thread2 = tree.threads.get('message2')
      assert.ok(thread2)
      assert.equal(thread2.get('parent'), '__root')
    })

    it('should recalculate descendant node count correctly', function() {
      var count = tree.calculateDescendantCount('message2')
      assert.deepEqual(count.toJS(), expectedMessage2Count)
    })

    it('should recalculate descendant node count with skip correctly', function() {
      var count = tree.calculateDescendantCount('message2', 1)
      assert.deepEqual(count.toJS(), {
        descendants: 1,
        newDescendants: 0,
        ownDescendants: 1,
        mentionDescendants: 0,
        newMentionDescendants: 0,
        latestDescendantTime: 22,
        latestDescendant: 'message2-2',
      })
    })

    describe('calling reset', function() {
      it('should empty the threads tree and return itself', function() {
        var ret = tree.reset()
        assert.equal(ret, tree)
        assert.equal(tree.threads.size, 0)
      })
    })
  })

  describe('after adding a messages as orphans', function() {
    var tree

    beforeEach(function() {
      tree = new ChatTree()

      var orphans = Immutable.Seq(testMessages)
        .map(message => {
          if (message.parent == '__root') {
            return _.extend({}, message, {parent: 'parent1-1-1'})
          } else {
            return message
          }
        })
        .toArray()
      tree.add(orphans)
    })

    it('should have the correct size', function() {
      // +1 for orphan parent
      assert.equal(tree.size, testMessages.length + 1)
    })

    it('should not calculate counts', function() {
      _.each(testMessages, entry => {
        assert.isNull(tree.getCount(entry.id))
      })
    })

    it('should not identify threads', function() {
      assert.equal(tree.threads.size, 0)
    })

    describe('and then adding the parents', function() {
      beforeEach(function() {
        tree.add([
          {
            '_seen': true,
            '_own': false,
            'id':'parent1',
            'parent': '__root',
            'time': 1,
            'content': 'woof!',
          },
          {
            '_seen': true,
            '_own': false,
            'id':'parent1-1',
            'parent': 'parent1',
            'time': 11,
            'content': 'bark!',
          },
          {
            '_seen': true,
            '_own': false,
            'id':'parent1-1-1',
            'parent': 'parent1-1',
            'time': 12,
            'content': 'meow!',
          },
        ])
      })

      it('should have the correct size', function() {
        assert.equal(tree.size, testMessages.length + 3)
      })

      it('should calculate correct counts', function() {
        checkMessage1Counts(tree)
        checkMessage2Counts(tree)

        assert.deepEqual(tree.getCount('parent1').toJS(), {
          descendants: 8,
          newDescendants: 1,
          ownDescendants: 3,
          mentionDescendants: 1,
          newMentionDescendants: 1,
          latestDescendantTime: 111,
          latestDescendant: 'message1-1-1',
        })

        assert.deepEqual(tree.getCount('parent1-1').toJS(), {
          descendants: 7,
          newDescendants: 1,
          ownDescendants: 3,
          mentionDescendants: 1,
          newMentionDescendants: 1,
          latestDescendantTime: 111,
          latestDescendant: 'message1-1-1',
        })

        assert.deepEqual(tree.getCount('parent1-1-1').toJS(), {
          descendants: 6,
          newDescendants: 1,
          ownDescendants: 3,
          mentionDescendants: 1,
          newMentionDescendants: 1,
          latestDescendantTime: 111,
          latestDescendant: 'message1-1-1',
        })
      })

      it('should identify threads', function() {
        assert.equal(tree.threads.size, 2)

        var parent11 = tree.threads.get('parent1-1-1')
        assert.ok(parent11)
        assert.equal(parent11.get('parent'), '__root')
        assert.deepEqual(parent11.get('children').toJS(), ['message2'])

        var thread2 = tree.threads.get('message2')
        assert.ok(thread2)
        assert.equal(thread2.get('parent'), 'parent1-1-1')
      })

      describe('and adding a message that creates a parent thread', function() {
        beforeEach(function() {
          tree.add([
            {
              '_seen': false,
              '_own': false,
              '_mention': true,
              'id':'parent1-2',
              'parent': 'parent1',
              'time': 123,
              'content': '@bark!',
            },
          ])
        })

        it('should calculate correct counts', function() {
          assert.deepEqual(tree.getCount('parent1').toJS(), {
            descendants: 9,
            newDescendants: 2,
            ownDescendants: 3,
            mentionDescendants: 2,
            newMentionDescendants: 2,
            latestDescendantTime: 123,
            latestDescendant: 'parent1-2',
          })
        })

        it('should identify the new thread and reparent the old one', function() {
          assert.equal(tree.threads.size, 3)

          var parent1 = tree.threads.get('parent1')
          assert.ok(parent1)
          assert.deepEqual(parent1.get('children').toJS(), ['parent1-1-1'])

          var parent111 = tree.threads.get('parent1-1-1')
          assert.ok(parent111)
          assert.equal(parent111.get('parent'), 'parent1')
        })

        describe('and adding a message that creates a thread in between parent and child threads', function() {
          beforeEach(function() {
            tree.add([
              {
                '_seen': false,
                '_own': false,
                '_mention': true,
                'id':'parent1-2-1',
                'parent': 'parent1-2',
                'time': 121,
                'content': '@howl one',
              },
              {
                '_seen': false,
                '_own': false,
                '_mention': true,
                'id':'parent1-2-2',
                'parent': 'parent1-2',
                'time': 121,
                'content': '@howl 2',
              },
              {
                '_seen': false,
                '_own': false,
                '_mention': true,
                'id':'parent1-1-2',
                'parent': 'parent1-1',
                'time': 112,
                'content': '@howl!',
              },
            ])
          })

          it('should reparent only the child thread', function() {
            assert.equal(tree.threads.size, 5)

            var parent12 = tree.threads.get('parent1-2')
            assert.ok(parent12)
            assert.equal(parent12.get('parent'), 'parent1')

            var parent11 = tree.threads.get('parent1-1')
            assert.ok(parent11)
            assert.equal(parent11.get('parent'), 'parent1')
            assert.deepEqual(parent11.get('children').toJS(), ['parent1-1-1'])

            var parent111 = tree.threads.get('parent1-1-1')
            assert.ok(parent111)
            assert.equal(parent111.get('parent'), 'parent1-1')
          })
        })

        describe('marking messages as read', function() {
          var parent1Score
          var parent11Score

          beforeEach(function() {
            parent1Score = tree.threads.get('parent1').get('score')
            parent11Score = tree.threads.get('parent1-1-1').get('score')
            tree.mergeNodes(['message1-1', 'parent1-2'], {_seen: 1000})
          })

          it('should not decrease thread scores', function() {
            assert.equal(tree.threads.get('parent1').get('score'), parent1Score)
            assert.equal(tree.threads.get('parent1-1-1').get('score'), parent11Score)
          })
        })
      })
    })
  })

  describe('getting the count of a nonexistent node', function() {
    it('should return null', function() {
      var tree = new ChatTree()
      assert.equal(tree.getCount('wat'), null)
    })
  })
})
