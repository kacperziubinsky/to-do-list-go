class UsersController < ApplicationController
  def register
    user = User.new(user_params)
    if user.save
      render json: { message: "User registered", user_id: user.id }, status: :created
    else
      render json: { errors: user.errors.full_messages }, status: :conflict
    end
  end

  def login
    user = User.find_by(username: params[:username])
    if user&.authenticate(params[:password])
      token = SecureRandom.hex(20)
      user.update(auth_token: token)
      render json: { token: token, username: user.username }
    else
      render json: { error: "Invalid credentials" }, status: :unauthorized
    end
  end

  private

  def user_params
    params.permit(:username, :password)
  end
end