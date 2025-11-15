#!/bin/bash

BASE_URL="http://localhost:8080"

echo "=== Заполнение базы тестовыми данными ==="

for team_num in {1..10}; do
    echo "Создание команды team_$team_num..."
    
    members='['
    for user_num in {1..20}; do
        user_id="user_${team_num}_${user_num}"
        username="User_${team_num}_${user_num}"
        
        members="$members{\"user_id\":\"$user_id\",\"username\":\"$username\",\"is_active\":true}"
        
        if [ $user_num -lt 20 ]; then
            members="$members,"
        fi
    done
    members="$members]"
    
    curl -s -X POST "$BASE_URL/team/add" \
      -H "Content-Type: application/json" \
      -d "{\"team_name\":\"team_$team_num\",\"members\":$members}" > /dev/null
    
    echo "✓ Команда team_$team_num создана (20 пользователей)"
done

echo ""
echo "=== Создание тестовых PR ==="

# Создаём 50 PR
for pr_num in {1..50}; do
    team_num=$((($pr_num % 10) + 1))
    author_num=$((($pr_num % 20) + 1))
    author_id="user_${team_num}_${author_num}"
    
    curl -s -X POST "$BASE_URL/pullRequest/create" \
      -H "Content-Type: application/json" \
      -d "{\"pull_request_id\":\"pr_test_$pr_num\",\"pull_request_name\":\"Test PR $pr_num\",\"author_id\":\"$author_id\"}" > /dev/null
    
    if [ $(($pr_num % 10)) -eq 0 ]; then
        echo "✓ Создано $pr_num PR"
    fi
done

echo ""
echo "=== Статистика ==="
curl -s "$BASE_URL/statistics" | python3 -m json.tool 2>/dev/null || curl -s "$BASE_URL/statistics"

echo ""
echo "✓ Данные успешно загружены!"
echo "  - Команд: 10"
echo "  - Пользователей: 200"
echo "  - PR: 50"