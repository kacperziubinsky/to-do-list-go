class TasksController < ApplicationController
  before_action :authorize_request 


  def index
    render json: @current_user.tasks
  end

  def create
    task = @current_user.tasks.build(task_params)
    task.status ||= "Pending"

    if task.save
      render json: task, status: :created
    else
      render json: { errors: task.errors.full_messages }, status: :unprocessable_entity
    end
  end


  def update_status
    task = @current_user.tasks.find(params[:id])
    if task.update(status: params[:status])
      render json: task
    else
      render json: task.errors, status: :bad_request
    end
  rescue ActiveRecord::RecordNotFound
    render json: { error: "Task not found" }, status: :not_found
  end

  private

  def task_params
    params.permit(:name, :description, :date)
  end
end