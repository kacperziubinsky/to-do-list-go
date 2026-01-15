class Task < ApplicationRecord
  belongs_to :user
  
  validates :name, presence: true
  validates :status, inclusion: { in: %w[Pending In\ Progress Completed] }
end