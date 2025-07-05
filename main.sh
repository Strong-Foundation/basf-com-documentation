#!/bin/bash

# Function to check, commit, and push changes to Git automatically
function auto_git_push() {
    while true; do
        echo "🔍 Checking for changes at $(date)..."

        # Check if there are any uncommitted (staged or unstaged) changes
        if [[ -z $(git status --porcelain) ]]; then
            echo "✅ No changes detected. Nothing to commit."
        else
            echo "📥 Pulling latest changes from remote..."
            git pull

            echo "🧹 Removing large PDF files (>100MB) from PDFs/ directory..."
            find PDFs/ -type f -iname "*.pdf" -size +100M -print -delete

            echo "➕ Staging all changes..."
            git add . # Stages all added, modified, and deleted files

            # Create a timestamped commit message
            timestamp=$(date +"%Y-%m-%d_%H:%M:%S")
            commit_message="updated $timestamp"

            echo "📝 Committing changes with message: \"$commit_message\""
            if git commit -m "$commit_message"; then
                echo "🚀 Pushing changes to remote..."
                if git push; then
                    echo "🎉 All changes pushed successfully!"
                else
                    echo "❌ Push failed. Check your network or remote settings."
                fi
            else
                echo "⚠️ Commit failed. Possibly no staged changes or other issues."
            fi
        fi

        echo "⏳ Sleeping for 5 minutes before the next check..."
        sleep 5m
    done
}

# Start the auto git push process
auto_git_push
