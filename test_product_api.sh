#!/bin/bash

# Product Management API Test Script
# This script tests all product and category endpoints

BASE_URL="http://localhost:8080"
COOKIE_FILE="cookies.txt"

echo "========================================="
echo "Product Management API Testing"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
PASS=0
FAIL=0

# Function to print test result
test_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        ((PASS++))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        ((FAIL++))
    fi
}

# Clean up previous cookie file
rm -f $COOKIE_FILE

echo "Step 1: Register a regular user"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"testpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Register user (expected 200, got $HTTP_CODE)"
echo ""

echo "Step 2: Register an admin user (will be promoted manually)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@example.com","password":"adminpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Register admin user (expected 200, got $HTTP_CODE)"
echo ""

echo "MANUAL STEP REQUIRED:"
echo "Run this command to promote the user to admin:"
echo "psql -d gess_staging -c \"UPDATE users SET role = 'admin' WHERE username = 'admin';\""
echo ""
read -p "Press Enter after promoting user to admin..."
echo ""

echo "Step 3: Login as admin"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"adminpass123"}' \
  -c $COOKIE_FILE)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Admin login (expected 200, got $HTTP_CODE)"
echo ""

echo "Step 4: List categories (public)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" $BASE_URL/categories)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "List categories (expected 200, got $HTTP_CODE)"
echo "Response: $BODY" | head -c 200
echo ""
echo ""

# Extract a category ID for testing
CATEGORY_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Using category ID: $CATEGORY_ID"
echo ""

echo "Step 5: Create a new product (Admin)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/products \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d "{
    \"name\": \"Test Laptop\",
    \"description\": \"High-performance test laptop\",
    \"price\": 999.99,
    \"stock_quantity\": 50,
    \"category_id\": \"$CATEGORY_ID\",
    \"image_url\": \"https://example.com/laptop.jpg\"
  }")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "201" ] && echo 0 || echo 1) "Create product (expected 201, got $HTTP_CODE)"
echo "Response: $BODY"
echo ""

# Extract product ID
PRODUCT_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Created product ID: $PRODUCT_ID"
echo ""

echo "Step 6: List all products (public)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?limit=5")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "List products (expected 200, got $HTTP_CODE)"
echo "Response: $BODY" | head -c 200
echo ""
echo ""

echo "Step 7: Get single product (public)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products/$PRODUCT_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Get product (expected 200, got $HTTP_CODE)"
echo "Response: $BODY"
echo ""

echo "Step 8: Search products by name (public)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?search=laptop")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Search products (expected 200, got $HTTP_CODE)"
echo ""

echo "Step 9: Filter products by price range (public)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?min_price=100&max_price=1500")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Filter by price (expected 200, got $HTTP_CODE)"
echo ""

echo "Step 10: Update product (Admin)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/admin/products/$PRODUCT_ID" \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d '{
    "name": "Updated Test Laptop",
    "price": 899.99,
    "stock_quantity": 45
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Update product (expected 200, got $HTTP_CODE)"
echo "Response: $BODY"
echo ""

echo "Step 11: Create a new category (Admin)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/categories \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d '{
    "name": "Test Category",
    "description": "A test category for API testing"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "201" ] && echo 0 || echo 1) "Create category (expected 201, got $HTTP_CODE)"
echo "Response: $BODY"
echo ""

# Extract new category ID
NEW_CATEGORY_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Created category ID: $NEW_CATEGORY_ID"
echo ""

echo "Step 12: Get category with products (public)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/categories/$CATEGORY_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Get category with products (expected 200, got $HTTP_CODE)"
echo ""

echo "Step 13: Update category (Admin)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/admin/categories/$NEW_CATEGORY_ID" \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d '{
    "name": "Updated Test Category",
    "description": "Updated description"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Update category (expected 200, got $HTTP_CODE)"
echo ""

echo "Step 14: Test non-admin access (should fail)"
echo "-------------------------------------------"
# Login as regular user
curl -s -X POST $BASE_URL/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass123"}' \
  -c cookies_user.txt > /dev/null

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/products \
  -H "Content-Type: application/json" \
  -b cookies_user.txt \
  -d '{
    "name": "Should Fail",
    "price": 100,
    "stock_quantity": 10
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "403" ] && echo 0 || echo 1) "Non-admin create product (expected 403, got $HTTP_CODE)"
rm -f cookies_user.txt
echo ""

echo "Step 15: Delete product (Admin)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/admin/products/$PRODUCT_ID" \
  -b $COOKIE_FILE)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Delete product (expected 200, got $HTTP_CODE)"
echo ""

echo "Step 16: Verify product is deleted"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products/$PRODUCT_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "404" ] && echo 0 || echo 1) "Get deleted product (expected 404, got $HTTP_CODE)"
echo ""

echo "Step 17: Delete category (Admin)"
echo "-------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/admin/categories/$NEW_CATEGORY_ID" \
  -b $COOKIE_FILE)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Delete category (expected 200, got $HTTP_CODE)"
echo ""

# Clean up
rm -f $COOKIE_FILE

echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "${GREEN}Passed: $PASS${NC}"
echo -e "${RED}Failed: $FAIL${NC}"
echo "Total: $((PASS + FAIL))"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
