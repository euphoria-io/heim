var assert = require('assert')
var Immutable = require('immutable')


describe('mention', function() {
  var mention = require('../lib/mention')

  describe('containsSubseq', function() {
    function assertContains(n, p, yes) {
      assert.equal(mention.containsSubseq(n, p), yes)
    }

    it('handles empty and exact matches', function() {
      assertContains("", "", true)
      assertContains("name", "", true)
      assertContains("", "cruft", false)
      assertContains("name", "name", true)
    })
    it('rejects extra on end of partial', function() {
      assertContains("name", "namex", false)
    })
    it('finds standard cases', function() {
      assertContains("name", "nm", true)
      assertContains("name", "e", true)
    })
    // This is a possible index bug.
    it('rejects extra chars going off the end', function() {
      assertContains("name", "es", false)
    })
    it('respects order', function() {
      assertContains("name", "eman", false)
    })
    it('respects count (always advance)', function() {
      assertContains("name", "nnm", false)
    })
  })

  describe('scoreMatch', function() {
    it('scores entirely bad matches as zero', function() {
      assert.equal(mention.scoreMatch('', ''), 0)
      assert.equal(mention.scoreMatch('something', ''), 0)
      assert.equal(mention.scoreMatch('', 'something'), 0)
      assert.equal(mention.scoreMatch('1', '2'), 0)
    })
    it('has expected score levels for a single name', function() {
      // Exact match scores same as prefix for now.
      assert.equal(mention.scoreMatch('TimMc', 'TimMc'), 31)
      assert.equal(mention.scoreMatch('TimMc', 'timmc'), 30)
      assert.equal(mention.scoreMatch('TimMc', 'Tim'), 31)
      assert.equal(mention.scoreMatch('TimMc', 'tim'), 30)
      assert.equal(mention.scoreMatch('TimMc', 'imM'), 21)
      assert.equal(mention.scoreMatch('TimMc', 'imm'), 20)
      assert.equal(mention.scoreMatch('TimMc', 'TM'), 11)
      assert.equal(mention.scoreMatch('TimMc', 'tm'), 10)
      assert.equal(mention.scoreMatch('TimMc', 'no'), 0)
    })
  })

  describe('rankCompletions', function() {
    function assertRanking(names, part, outNames) {
      var actual = mention.rankCompletions(Immutable.fromJS(names), part);
      assert.equal(actual, outNames.toArray());
    }
    var users = ["chromakode" "logan" "mac" "Max" "TimMc"]

    it('puts prefix over infix, tie-breaks with case, and is stable', function() {
      assertRanking(users, "M", ["Max" "mac" "TimMc" "chromakode"])
    })
    it('ranks subseqs less than infix ci, and is stable', function() {
      assertRanking(users, 'mc', ["TimMc" "mac"])
    })
  })
})
