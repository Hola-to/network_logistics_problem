#!/bin/bash
# scripts/fix_migrations.sh

MIGRATIONS_DIR="migrations/postgres"

echo "Fixing migration files..."

# Найти все .up.sql файлы
for up_file in "$MIGRATIONS_DIR"/*.up.sql; do
    if [ -f "$up_file" ]; then
        # Получить базовое имя (без .up.sql)
        base_name="${up_file%.up.sql}"
        down_file="${base_name}.down.sql"
        new_file="${base_name}.sql"

        echo "Processing: $up_file"

        # Создать объединённый файл
        echo "-- +goose Up" > "$new_file"
        cat "$up_file" >> "$new_file"
        echo "" >> "$new_file"

        if [ -f "$down_file" ]; then
            echo "-- +goose Down" >> "$new_file"
            cat "$down_file" >> "$new_file"
            rm "$down_file"
            echo "  Merged with: $down_file"
        else
            echo "-- +goose Down" >> "$new_file"
            echo "-- TODO: Add rollback SQL" >> "$new_file"
            echo "  Warning: No down file found"
        fi

        rm "$up_file"
        echo "  Created: $new_file"
    fi
done

echo "Done!"
