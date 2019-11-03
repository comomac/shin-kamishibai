# frozen_string_literal: true

# Based on Dave Koelle's Alphanum algorithm
# Rik Hemsley, 2007
# See also http://rikkus.info/arch/sensible_sort.rb

#
# Released under the MIT License - https://opensource.org/licenses/MIT
#
# Permission is hereby granted, free of charge, to any person obtaining
# a copy of this software and associated documentation files (the "Software"),
# to deal in the Software without restriction, including without limitation
# the rights to use, copy, modify, merge, publish, distribute, sublicense,
# and/or sell copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
# IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
# DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
# OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE
# USE OR OTHER DEALINGS IN THE SOFTWARE.
#

require 'test/unit'

module Enumerable
  def sensible_sort
    sort { |a, b| grouped_compare(a, b) }
  end

  def sensible_sort!
    sort! { |a, b| grouped_compare(a, b) }
  end

  private

  def grouped_compare(a, b)
    loop do
      a_chunk, a = extract_alpha_or_number_group(a)
      b_chunk, b = extract_alpha_or_number_group(b)

      ret = a_chunk <=> b_chunk

      return -1 if a_chunk == ''
      return ret if ret != 0
    end
  end

  def extract_alpha_or_number_group(item)
    matchdata = /([A-Za-z]+|[\d]+)/.match(item)

    if matchdata.nil?
      ['', '']
    else
      [matchdata[0], item = item[matchdata.offset(0)[1]..-1]]
    end
  end
end

if $PROGRAM_NAME == __FILE__

  class AlphanumericSortTestCase < Test::Unit::TestCase
    def test_empty
      assert_equal(['', ''], ['', ''].sensible_sort)
    end

    def test_identical_simple
      assert_equal(%w[x x], %w[x x].sensible_sort)
    end

    def test_identical_two_groups
      assert_equal(%w[x1 x1], %w[x1 x1].sensible_sort)
    end

    def test_ordered_simple
      assert_equal(%w[x y], %w[x y].sensible_sort)
    end

    def test_ordered_simple_start_backwards
      assert_equal(%w[x y], %w[y x].sensible_sort)
    end

    def test_ordered_two_groups
      assert_equal(%w[x1 x2], %w[x1 x2].sensible_sort)
    end

    def test_ordered_two_groups_start_backwards
      assert_equal(%w[x1 x2], %w[x2 x1].sensible_sort)
    end

    def test_ordered_two_groups_separated
      assert_equal(%w[x_1 x_2], %w[x_2 x_1].sensible_sort)
    end

    def test_ordered_two_groups_separated_different_distances
      assert_equal(%w[x_1 x__2], %w[x__2 x_1].sensible_sort)
    end

    def test_ordered_two_groups_separated_different_distances_swapped
      assert_equal(%w[x__1 x_2], %w[x_2 x__1].sensible_sort)
    end

    def test_three_groups
      assert_equal(
        ['hello 2 world', 'hello world', 'hello world 2'],
        ['hello world', 'hello world 2', 'hello 2 world'].sensible_sort
      )
    end

    def test!
      x = ['hello world', 'hello world 2', 'hello 2 world']
      x.sensible_sort!
      assert_equal(['hello 2 world', 'hello world', 'hello world 2'], x)
    end
  end

end
