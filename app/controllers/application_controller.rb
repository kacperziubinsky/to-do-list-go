class ApplicationController < ActionController::API
  def authorize_request
    header = request.headers['Authorization']
    token = header.split(' ').last if header
    @current_user = User.find_by(auth_token: token)

    render json: { error: 'Unauthorized' }, status: :unauthorized unless @current_user
  end
end