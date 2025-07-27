#include <iostream>
#include <nlohmann/json.hpp>
#include <filesystem>
#include <cmath>
#include "JsonParser.hpp"

using JSON = nlohmann::json;
using namespace std;

void Writescore(double score){
    const string file = "score.txt";
    double old_score = 0.0;
    if (std::filesystem::exists(file)) {
        std::ifstream infile(file);
        std::string line;
        if (std::getline(infile, line)) {
            std::istringstream iss(line);
            iss >> old_score;
        }
        infile.close();
    }
    double new_score = max(score,old_score);
    if(new_score != old_score){
        std::ofstream outfile(file);
        outfile << std::fixed << std::setprecision(2) << new_score << std::endl;
        outfile.close();
    }
}
void WriteJsonfile(string json_path){
    const string file = "message.txt";
    std::string input_path = json_path;
    std::ifstream input_file(input_path);
    if (!input_file) {
        std::cerr << "Failed to open input file: " << input_path << std::endl;
    }

    std::ofstream output_file("message.txt");
    if (!output_file) {
        std::cerr << "Failed to open output file: message.txt" << std::endl;
    }

    output_file << input_file.rdbuf(); 

    input_file.close();
    output_file.close();

    std::cout << "Copied content from " << input_path << " to message.txt" << std::endl;
}

int main(int argc, char* argv[]) {
    if (argc != 3) {
        cerr << "Usage: ./grp_parser <gtest.json> <score.json>" << "\n";
    }
    JSONParser parser(argv[1],argv[2]);
    parser.Parse();
    cout<< parser.Getscore() <<"\n";
    Writescore(parser.Getscore());
    WriteJsonfile(argv[1]);

    return 0;
}
