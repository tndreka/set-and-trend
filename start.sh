#!/bin/bash

# Set and Trend - Docker Startup Script
# This script helps you run the project on any machine

set -e

echo "🚀 Set and Trend - Docker Setup"
echo "================================"
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first:"
    echo "   https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Docker Compose is available
if ! docker compose version &> /dev/null && ! docker-compose version &> /dev/null; then
    echo "❌ Docker Compose is not available. Please install Docker Compose."
    exit 1
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file from example..."
    cp .env.example .env
    echo "⚠️  Please edit .env file with your secure passwords before running in production!"
    echo ""
fi

# Parse command line arguments
case "${1:-up}" in
    up|start)
        echo "🐳 Starting all services..."
        docker compose up -d
        echo ""
        echo "✅ Services started!"
        echo ""
        echo "📊 Access the application:"
        echo "   Frontend:  http://localhost:3000"
        echo "   Backend:   http://localhost:8080"
        echo "   Database:  localhost:5432"
        echo ""
        echo "📋 View logs: ./start.sh logs"
        echo "🛑 Stop:      ./start.sh down"
        ;;
    
    down|stop)
        echo "🛑 Stopping all services..."
        docker compose down
        echo "✅ Services stopped!"
        ;;
    
    restart)
        echo "🔄 Restarting all services..."
        docker compose down
        docker compose up -d
        echo "✅ Services restarted!"
        ;;
    
    build)
        echo "🔨 Building all services..."
        docker compose build --no-cache
        echo "✅ Build complete!"
        ;;
    
    logs)
        if [ -z "$2" ]; then
            docker compose logs -f
        else
            docker compose logs -f "$2"
        fi
        ;;
    
    status)
        echo "📊 Service Status:"
        docker compose ps
        ;;
    
    clean)
        echo "🧹 Cleaning up..."
        docker compose down -v --rmi local
        echo "✅ Cleanup complete!"
        echo "⚠️  Note: This removed all containers, volumes, and local images."
        ;;
    
    migrate)
        echo "📦 Running database migrations..."
        docker compose exec postgres psql -U stt_user -d set_the_trend -f /docker-entrypoint-initdb.d/000001_init_schema.up.sql
        docker compose exec postgres psql -U stt_user -d set_the_trend -f /docker-entrypoint-initdb.d/000006_add_user_auth_fields.up.sql
        echo "✅ Migrations complete!"
        ;;
    
    seed)
        echo "🌱 Seeding database with sample data..."
        echo "⚠️  Run the import scripts manually after containers are up:"
        echo "   docker compose exec backend ./scripts/import_csv/main"
        echo "   docker compose exec backend ./scripts/compute_all_emas/main"
        echo "   docker compose exec backend ./scripts/evaluate_all_rules/main"
        ;;
    
    shell-backend)
        echo "🐚 Opening shell in backend container..."
        docker compose exec backend sh
        ;;
    
    shell-db)
        echo "🐚 Opening psql in database container..."
        docker compose exec postgres psql -U stt_user -d set_the_trend
        ;;
    
    *)
        echo "Usage: ./start.sh [command]"
        echo ""
        echo "Commands:"
        echo "  up, start     Start all services (default)"
        echo "  down, stop    Stop all services"
        echo "  restart       Restart all services"
        echo "  build         Rebuild all containers"
        echo "  logs [svc]    View logs (optionally for specific service)"
        echo "  status        Show service status"
        echo "  clean         Remove all containers, volumes, and images"
        echo "  migrate       Run database migrations"
        echo "  seed          Instructions for seeding data"
        echo "  shell-backend Open shell in backend container"
        echo "  shell-db      Open psql in database container"
        ;;
esac
