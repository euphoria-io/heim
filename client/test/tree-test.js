require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')
var Immutable = require('immutable')

var Tree = require('../lib/tree')

describe('Tree', function() {
  function debugMap(node, children, depth) {
    return [node.get('id'), depth, children.toArray()]
  }

  function checkMapOnlyRoot(tree) {
    assert.deepEqual(tree.mapDFS(debugMap), ['__root', 0, []])
  }

  function expectEmit(tree, ids) {
    sinon.assert.callCount(tree.changes.emit, ids.length + 1)
    Immutable.Seq(ids).forEach(function(id) {
      sinon.assert.calledWithExactly(tree.changes.emit, id, tree.get(id))
    })
    sinon.assert.calledWithExactly(tree.changes.emit, '__all', ids)
  }

  describe('a new empty tree', function() {
    var tree = new Tree()

    it('should have size 0', function() {
      assert.equal(tree.size, 0)
    })

    it('should only visit root in traversal', function() {
      checkMapOnlyRoot(tree)
    })
  })

  describe('initialized with an array of entries', function() {
    var entries = [
      {id: '1', value: 'hello', time: 5},
      {id: '2', parent: '1', value: 'world', time: 5},
    ]
    var tree

    beforeEach(function() {
      tree = new Tree('time')
      sinon.stub(tree.changes, 'emit')
      tree.reset(entries)
    })

    afterEach(function() {
      tree.changes.emit.restore()
    })

    it('should have correct size', function() {
      assert.equal(tree.size, 2)
    })

    it('should nest nodes with parents', function() {
      assert(tree.get('1').get('children').contains('2'))
    })

    it('should trigger a change event on the new nodes and root', function() {
      expectEmit(tree, ['1', '2', '__root'])
    })

    it('should visit all nodes in a map traversal', function() {
      var visited = tree.mapDFS(debugMap)
      assert.deepEqual(visited, [
        '__root', 0, [
          ['1', 1, [
            ['2', 2, []]
          ]]
        ]
      ])
    })

    describe('after adding a node', function() {
      beforeEach(function() {
        tree.changes.emit.reset()
        tree.add({id: '3', parent: '1', value: 'yo', time: 7})
      })

      it('should contain the new node', function() {
        assert(tree.get('1').get('children').contains('3'))
      })

      it('should update the size', function() {
        assert.equal(tree.size, 3)
      })

      it('should trigger a change event on the new node and parent', function() {
        expectEmit(tree, ['3', '1'])
      })

      it('the new node should be last', function() {
        assert.equal(tree.last(), tree.get('3'))
      })

      it('should visit all nodes in a map traversal', function() {
        var visited = tree.mapDFS(debugMap)
        assert.deepEqual(visited, [
          '__root', 0, [
            ['1', 1, [
              ['2', 2, []],
              ['3', 2, []],
            ]]
          ]
        ])
      })
    })

    describe('after adding multiple nodes', function() {
      var entries = [
        {id: '2', parent: '1', value: 'world', time: 5},
        {id: '0', value: 'first!', time: 0},
        {id: '3', parent: '1', value: 'local first!', time: 1},
        {id: '9', value: 'last', time: 9},
      ]

      beforeEach(function() {
        tree.changes.emit.reset()
        tree.add(entries)
      })

      function check() {
        it('the size should be correct', function() {
          assert.equal(tree.size, 5)
        })

        it('the node with highest time should be last', function() {
          assert.equal(tree.last(), tree.get('9'))
        })

        it('should visit all nodes in a map traversal (in the right order)', function() {
          var visited = tree.mapDFS(debugMap)
          assert.deepEqual(visited, [
            '__root', 0, [
              ['0', 1, []],
              ['1', 1, [
                ['3', 2, []],
                ['2', 2, []],
              ]],
              ['9', 1, []],
            ]
          ])
        })
      }

      check()

      it('should only trigger a change event for new nodes and the parents of new nodes', function() {
        expectEmit(tree, ['0', '3', '9', '1', '__root'])
      })

      describe('after re-adding the same nodes', function() {
        beforeEach(function() {
          tree.changes.emit.reset()
          tree.add(entries)
        })

        describe('should not change', function() {
          check()
        })

        it('should not trigger a change event', function() {
          sinon.assert.notCalled(tree.changes.emit)
        })
      })
    })

    describe('after adding a node with a missing parent', function() {
      beforeEach(function() {
        tree.changes.emit.reset()
        tree.add({id: '3', parent: 'wtf', value: 'yo', time: 7})
      })

      it('should contain the new node', function() {
        assert(tree.get('3'))
      })

      it('should contain an unreachable parent for the new node', function() {
        var parent = tree.get('wtf')
        assert(parent.get('children').contains('3'))
        assert(!parent.has('parent'))
      })

      it('should update the size', function() {
        assert.equal(tree.size, 4)
      })

      describe('after adding the missing parent', function() {
        beforeEach(function() {
          tree.changes.emit.reset()
          tree.add({id: 'wtf', parent: '1', value: 'j0', time: 6})
        })

        it('should update the size', function() {
          assert.equal(tree.size, 4)
        })

        it('should trigger a change event on the child and parent', function() {
          expectEmit(tree, ['wtf', '1'])
        })

        it('should visit all nodes in a map traversal', function() {
          var visited = tree.mapDFS(debugMap)
          assert.deepEqual(visited, [
            '__root', 0, [
              ['1', 1, [
                ['2', 2, []],
                ['wtf', 2, [
                  ['3', 3, []]
                ]],
              ]]
            ]
          ])
        })
      })
    })

    describe('after merging an update to a node', function() {
      it('should update the node', function() {
        tree.mergeNode('2', {value: 'dawg'})
        assert.equal(tree.get('2').get('value'), 'dawg')
      })

      it('should retain the same size', function() {
        tree.mergeNode('2', {value: 'dawg'})
        assert.equal(tree.size, 2)
      })

      it('should trigger a change event', function() {
        tree.changes.emit.reset()
        tree.mergeNode('2', {value: 'dawg'})
        expectEmit(tree, ['2'])
      })

      it('should not trigger a change event if unchanged', function() {
        tree.changes.emit.reset()
        tree.mergeNode('2', {value: 'world'})
        sinon.assert.notCalled(tree.changes.emit)
      })
    })

    describe('after resetting to empty', function() {
      beforeEach(function() {
        tree.changes.emit.reset()
        tree.reset()
      })

      it('should be empty', function() {
        assert.equal(tree.size, 0)
      })

      it('should only visit root in traversal', function() {
        checkMapOnlyRoot(tree)
      })

      it('should trigger a change event', function() {
        expectEmit(tree, ['__root'])
      })
    })
  })
})
