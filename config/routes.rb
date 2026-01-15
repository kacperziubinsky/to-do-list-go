Rails.application.routes.draw do
  post '/register', to: 'users#register'
  post '/login', to: 'users#login'
  
  resources :tasks, only: [:index, :create] do
    member do
      patch 'complete', to: 'tasks#update_status', status: 'Completed'
      patch 'in-progress', to: 'tasks#update_status', status: 'In Progress'
      patch 'pending', to: 'tasks#update_status', status: 'Pending'
    end
  end
end