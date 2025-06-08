#!/bin/bash

# Test runner script for OJ-API
# This script runs all tests and provides a comprehensive testing report

echo "Starting OJ-API Test Suite..."
echo "================================="

# Set test environment
export GIN_MODE=test
export GO_ENV=test

# Function to run tests for a specific package
run_package_tests() {
    local package=$1
    echo ""
    echo "Testing package: $package"
    echo "--------------------------------"
    
    # Run tests with verbose output and coverage
    go test -v -race -coverprofile="${package//\//_}_coverage.out" "./$package"
    
    if [ $? -eq 0 ]; then
        echo "✅ $package tests PASSED"
    else
        echo "❌ $package tests FAILED"
        return 1
    fi
}

# Function to run all tests
run_all_tests() {
    echo "Running all tests..."
    
    # Test packages
    packages=("routes" "handlers")
    failed_packages=()
    
    for package in "${packages[@]}"; do
        if ! run_package_tests "$package"; then
            failed_packages+=("$package")
        fi
    done
    
    echo ""
    echo "================================="
    echo "Test Summary"
    echo "================================="
    
    if [ ${#failed_packages[@]} -eq 0 ]; then
        echo "🎉 All tests PASSED!"
    else
        echo "❌ Some tests FAILED:"
        for package in "${failed_packages[@]}"; do
            echo "  - $package"
        done
        return 1
    fi
}

# Function to generate coverage report
generate_coverage_report() {
    echo ""
    echo "Generating coverage report..."
    
    # Combine coverage files
    echo "mode: atomic" > coverage.out
    find . -name "*_coverage.out" -exec grep -h -v "mode: atomic" {} \; >> coverage.out
    
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    
    # Show coverage summary
    go tool cover -func=coverage.out | tail -1
    
    echo "Coverage report generated: coverage.html"
}

# Function to run specific test patterns
run_test_pattern() {
    local pattern=$1
    echo "Running tests matching pattern: $pattern"
    go test -v -run "$pattern" ./...
}

# Function to run benchmarks
run_benchmarks() {
    echo "Running benchmarks..."
    go test -bench=. -benchmem ./...
}

# Main execution
case "${1:-all}" in
    "all")
        run_all_tests
        if [ $? -eq 0 ]; then
            generate_coverage_report
        fi
        ;;
    "coverage")
        run_all_tests
        generate_coverage_report
        ;;
    "routes")
        run_package_tests "routes"
        ;;
    "handlers")
        run_package_tests "handlers"
        ;;
    "admin")
        run_test_pattern "Admin"
        ;;
    "exam")
        run_test_pattern "Exam"
        ;;
    "question")
        run_test_pattern "Question"
        ;;
    "score")
        run_test_pattern "Score"
        ;;
    "user")
        run_test_pattern "User"
        ;;
    "gitea")
        run_test_pattern "Gitea"
        ;;
    "sandbox")
        run_test_pattern "Sandbox"
        ;;
    "webhook")
        run_test_pattern "Webhook"
        ;;
    "bench")
        run_benchmarks
        ;;
    "clean")
        echo "Cleaning test artifacts..."
        rm -f *_coverage.out coverage.out coverage.html
        echo "Cleaned!"
        ;;
    *)
        echo "Usage: $0 [all|coverage|routes|handlers|admin|exam|question|score|user|gitea|sandbox|webhook|bench|clean]"
        echo ""
        echo "Options:"
        echo "  all       - Run all tests (default)"
        echo "  coverage  - Run all tests and generate coverage report"
        echo "  routes    - Run only routes tests"
        echo "  handlers  - Run only handlers tests"
        echo "  admin     - Run only admin-related tests"
        echo "  exam      - Run only exam-related tests"
        echo "  question  - Run only question-related tests"
        echo "  score     - Run only score-related tests"
        echo "  user      - Run only user-related tests"
        echo "  gitea     - Run only gitea-related tests"
        echo "  sandbox   - Run only sandbox-related tests"
        echo "  webhook   - Run only webhook-related tests"
        echo "  bench     - Run benchmarks"
        echo "  clean     - Clean test artifacts"
        ;;
esac
