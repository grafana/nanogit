# 🚀 Performance Benchmark Report

**Generated:** 2025-07-08T11:47:42+02:00  
**Total Benchmarks:** 168

## 📊 Performance Overview

| Operation                              | Speed Winner | Duration | In-Memory Winner | Memory Usage |
| -------------------------------------- | ------------ | -------- | ---------------- | ------------ |
| BulkCreateFiles_bulk_1000_files_medium | 🚀 nanogit    | 892.4ms  | 💚 nanogit        | 7.7 MB       |
| BulkCreateFiles_bulk_1000_files_small  | 🚀 nanogit    | 846.5ms  | 💚 nanogit        | 6.0 MB       |
| BulkCreateFiles_bulk_100_files_medium  | 🚀 nanogit    | 99.9ms   | 💚 nanogit        | 4.4 MB       |
| BulkCreateFiles_bulk_100_files_small   | 🚀 nanogit    | 93.6ms   | 💚 nanogit        | 3.5 MB       |
| CompareCommits_adjacent_commits_large  | 🚀 nanogit    | 118.8ms  | 💚 nanogit        | 6.7 MB       |
| CompareCommits_adjacent_commits_medium | 🚀 nanogit    | 103.2ms  | 💚 nanogit        | 2.8 MB       |
| CompareCommits_adjacent_commits_small  | 🐹 go-git     | 74.3ms   | 💚 nanogit        | 1.4 MB       |
| CompareCommits_adjacent_commits_xlarge | 🚀 nanogit    | 165.6ms  | 💚 nanogit        | 17.4 MB      |
| CompareCommits_few_commits_large       | 🚀 nanogit    | 188.4ms  | 💚 nanogit        | 6.9 MB       |
| CompareCommits_few_commits_medium      | 🚀 nanogit    | 174.4ms  | 💚 nanogit        | 3.5 MB       |
| CompareCommits_few_commits_small       | 🐹 go-git     | 86.8ms   | 💚 nanogit        | 2.1 MB       |
| CompareCommits_few_commits_xlarge      | 🚀 nanogit    | 260.7ms  | 💚 nanogit        | 17.4 MB      |
| CompareCommits_max_commits_large       | 🚀 nanogit    | 301.8ms  | 💚 nanogit        | 7.4 MB       |
| CompareCommits_max_commits_medium      | 🚀 nanogit    | 264.3ms  | 💚 nanogit        | 4.2 MB       |
| CompareCommits_max_commits_small       | 🐹 go-git     | 94.0ms   | 💚 nanogit        | 2.9 MB       |
| CompareCommits_max_commits_xlarge      | 🚀 nanogit    | 364.6ms  | 💚 nanogit        | 18.2 MB      |
| CreateFile_large_repo                  | 🚀 nanogit    | 61.5ms   | 💚 nanogit        | 3.7 MB       |
| CreateFile_medium_repo                 | 🚀 nanogit    | 53.7ms   | 💚 nanogit        | 1.5 MB       |
| CreateFile_small_repo                  | 🚀 nanogit    | 59.6ms   | 💚 nanogit        | 1.5 MB       |
| CreateFile_xlarge_repo                 | 🚀 nanogit    | 86.3ms   | 💚 nanogit        | 12.6 MB      |
| DeleteFile_large_repo                  | 🚀 nanogit    | 59.0ms   | 💚 nanogit        | 3.4 MB       |
| DeleteFile_medium_repo                 | 🚀 nanogit    | 51.0ms   | 💚 nanogit        | 1.4 MB       |
| DeleteFile_small_repo                  | 🚀 nanogit    | 52.0ms   | 💚 nanogit        | 1.4 MB       |
| DeleteFile_xlarge_repo                 | 🚀 nanogit    | 81.1ms   | 💚 nanogit        | 12.7 MB      |
| GetFlatTree_large_tree                 | 🚀 nanogit    | 52.9ms   | 💚 nanogit        | 4.0 MB       |
| GetFlatTree_medium_tree                | 🚀 nanogit    | 39.6ms   | 💚 nanogit        | 1.3 MB       |
| GetFlatTree_small_tree                 | 🚀 nanogit    | 52.2ms   | 💚 nanogit        | 677.2 KB     |
| GetFlatTree_xlarge_tree                | 🚀 nanogit    | 78.5ms   | 💚 nanogit        | 11.3 MB      |
| UpdateFile_large_repo                  | 🚀 nanogit    | 71.2ms   | 💚 nanogit        | 3.6 MB       |
| UpdateFile_medium_repo                 | 🚀 nanogit    | 52.6ms   | 💚 nanogit        | 1.4 MB       |
| UpdateFile_small_repo                  | 🚀 nanogit    | 48.3ms   | 💚 nanogit        | 1.4 MB       |
| UpdateFile_xlarge_repo                 | 🚀 nanogit    | 81.3ms   | 💚 nanogit        | 12.8 MB      |

## ⚡ Duration Comparison

| Operation                              | git-cli   | go-git    | nanogit   |
| -------------------------------------- | --------- | --------- | --------- |
| BulkCreateFiles_bulk_1000_files_medium | 10.93s 🐌  | 70.61s 🐌  | 892.4ms 🏆 |
| BulkCreateFiles_bulk_1000_files_small  | 9.66s 🐌   | 19.25s 🐌  | 846.5ms 🏆 |
| BulkCreateFiles_bulk_100_files_medium  | 2.15s 🐌   | 6.38s 🐌   | 99.9ms 🏆  |
| BulkCreateFiles_bulk_100_files_small   | 1.60s 🐌   | 798.8ms 🐌 | 93.6ms 🏆  |
| CompareCommits_adjacent_commits_large  | 1.26s 🐌   | 2.64s 🐌   | 118.8ms 🏆 |
| CompareCommits_adjacent_commits_medium | 659.3ms 🐌 | 442.2ms   | 103.2ms 🏆 |
| CompareCommits_adjacent_commits_small  | 528.6ms 🐌 | 74.3ms 🏆  | 81.0ms ✅  |
| CompareCommits_adjacent_commits_xlarge | 5.37s 🐌   | 20.17s 🐌  | 165.6ms 🏆 |
| CompareCommits_few_commits_large       | 1.28s 🐌   | 2.57s 🐌   | 188.4ms 🏆 |
| CompareCommits_few_commits_medium      | 655.4ms   | 400.1ms   | 174.4ms 🏆 |
| CompareCommits_few_commits_small       | 523.1ms 🐌 | 86.8ms 🏆  | 158.5ms ✅ |
| CompareCommits_few_commits_xlarge      | 5.68s 🐌   | 20.10s 🐌  | 260.7ms 🏆 |
| CompareCommits_max_commits_large       | 1.29s     | 2.58s 🐌   | 301.8ms 🏆 |
| CompareCommits_max_commits_medium      | 670.8ms   | 468.2ms ✅ | 264.3ms 🏆 |
| CompareCommits_max_commits_small       | 529.3ms 🐌 | 94.0ms 🏆  | 244.8ms   |
| CompareCommits_max_commits_xlarge      | 5.80s 🐌   | 20.17s 🐌  | 364.6ms 🏆 |
| CreateFile_large_repo                  | 1.99s 🐌   | 2.86s 🐌   | 61.5ms 🏆  |
| CreateFile_medium_repo                 | 1.35s 🐌   | 516.2ms 🐌 | 53.7ms 🏆  |
| CreateFile_small_repo                  | 1.18s 🐌   | 121.6ms   | 59.6ms 🏆  |
| CreateFile_xlarge_repo                 | 6.49s 🐌   | 22.27s 🐌  | 86.3ms 🏆  |
| DeleteFile_large_repo                  | 2.05s 🐌   | 2.85s 🐌   | 59.0ms 🏆  |
| DeleteFile_medium_repo                 | 1.34s 🐌   | 495.5ms 🐌 | 51.0ms 🏆  |
| DeleteFile_small_repo                  | 1.16s 🐌   | 116.2ms   | 52.0ms 🏆  |
| DeleteFile_xlarge_repo                 | 6.50s 🐌   | 22.54s 🐌  | 81.1ms 🏆  |
| GetFlatTree_large_tree                 | 1.34s 🐌   | 2.61s 🐌   | 52.9ms 🏆  |
| GetFlatTree_medium_tree                | 713.2ms 🐌 | 443.7ms 🐌 | 39.6ms 🏆  |
| GetFlatTree_small_tree                 | 542.9ms 🐌 | 75.9ms ✅  | 52.2ms 🏆  |
| GetFlatTree_xlarge_tree                | 5.36s 🐌   | 19.84s 🐌  | 78.5ms 🏆  |
| UpdateFile_large_repo                  | 2.00s 🐌   | 2.86s 🐌   | 71.2ms 🏆  |
| UpdateFile_medium_repo                 | 1.36s 🐌   | 499.0ms 🐌 | 52.6ms 🏆  |
| UpdateFile_small_repo                  | 1.16s 🐌   | 113.9ms   | 48.3ms 🏆  |
| UpdateFile_xlarge_repo                 | 6.78s 🐌   | 22.46s 🐌  | 81.3ms 🏆  |

## 💾 Memory Usage Comparison

*Note: git-cli uses disk storage rather than keeping data in memory, so memory comparisons focus on in-memory clients (nanogit vs go-git)*

| Operation                              | git-cli      | go-git     | nanogit    |
| -------------------------------------- | ------------ | ---------- | ---------- |
| BulkCreateFiles_bulk_1000_files_medium | -5946528 B 💾 | 61.0 MB 🔥  | 7.7 MB 🏆   |
| BulkCreateFiles_bulk_1000_files_small  | -1048968 B 💾 | 15.4 MB    | 6.0 MB 🏆   |
| BulkCreateFiles_bulk_100_files_medium  | 5.1 MB 💾     | 54.1 MB 🔥  | 4.4 MB 🏆   |
| BulkCreateFiles_bulk_100_files_small   | 5.1 MB 💾     | 8.9 MB     | 3.5 MB 🏆   |
| CompareCommits_adjacent_commits_large  | 70.5 KB 💾    | 227.1 MB 🔥 | 6.7 MB 🏆   |
| CompareCommits_adjacent_commits_medium | 70.2 KB 💾    | 39.6 MB 🔥  | 2.8 MB 🏆   |
| CompareCommits_adjacent_commits_small  | 70.2 KB 💾    | 5.7 MB     | 1.4 MB 🏆   |
| CompareCommits_adjacent_commits_xlarge | 70.2 KB 💾    | 1.4 GB 🔥   | 17.4 MB 🏆  |
| CompareCommits_few_commits_large       | 70.5 KB 💾    | 178.5 MB 🔥 | 6.9 MB 🏆   |
| CompareCommits_few_commits_medium      | 70.2 KB 💾    | 43.2 MB 🔥  | 3.5 MB 🏆   |
| CompareCommits_few_commits_small       | 70.5 KB 💾    | 6.5 MB     | 2.1 MB 🏆   |
| CompareCommits_few_commits_xlarge      | 70.2 KB 💾    | 1.6 GB 🔥   | 17.4 MB 🏆  |
| CompareCommits_max_commits_large       | 70.5 KB 💾    | 223.1 MB 🔥 | 7.4 MB 🏆   |
| CompareCommits_max_commits_medium      | 70.2 KB 💾    | 42.2 MB 🔥  | 4.2 MB 🏆   |
| CompareCommits_max_commits_small       | 70.5 KB 💾    | 7.2 MB     | 2.9 MB 🏆   |
| CompareCommits_max_commits_xlarge      | 70.2 KB 💾    | 1.6 GB 🔥   | 18.2 MB 🏆  |
| CreateFile_large_repo                  | 135.7 KB 💾   | 273.0 MB 🔥 | 3.7 MB 🏆   |
| CreateFile_medium_repo                 | 136.7 KB 💾   | 33.9 MB 🔥  | 1.5 MB 🏆   |
| CreateFile_small_repo                  | 136.3 KB 💾   | 4.4 MB     | 1.5 MB 🏆   |
| CreateFile_xlarge_repo                 | 135.8 KB 💾   | 1.9 GB 🔥   | 12.6 MB 🏆  |
| DeleteFile_large_repo                  | 135.8 KB 💾   | 273.0 MB 🔥 | 3.4 MB 🏆   |
| DeleteFile_medium_repo                 | 135.7 KB 💾   | 44.3 MB 🔥  | 1.4 MB 🏆   |
| DeleteFile_small_repo                  | 136.1 KB 💾   | 3.8 MB     | 1.4 MB 🏆   |
| DeleteFile_xlarge_repo                 | 135.6 KB 💾   | 2.0 GB 🔥   | 12.7 MB 🏆  |
| GetFlatTree_large_tree                 | 3.2 MB 💾     | 241.1 MB 🔥 | 4.0 MB 🏆   |
| GetFlatTree_medium_tree                | 740.5 KB 💾   | 44.4 MB 🔥  | 1.3 MB 🏆   |
| GetFlatTree_small_tree                 | 154.6 KB 💾   | 4.2 MB 🔥   | 677.2 KB 🏆 |
| GetFlatTree_xlarge_tree                | 18.7 MB 💾    | 1.6 GB 🔥   | 11.3 MB 🏆  |
| UpdateFile_large_repo                  | 135.2 KB 💾   | 271.9 MB 🔥 | 3.6 MB 🏆   |
| UpdateFile_medium_repo                 | 135.5 KB 💾   | 27.9 MB 🔥  | 1.4 MB 🏆   |
| UpdateFile_small_repo                  | 135.2 KB 💾   | 4.0 MB     | 1.4 MB 🏆   |
| UpdateFile_xlarge_repo                 | 135.1 KB 💾   | 2.0 GB 🔥   | 12.8 MB 🏆  |

## 🎯 Nanogit Performance Analysis

### ⚡ Speed Comparison

| Operation                              | vs git-cli     | vs go-git       |
| -------------------------------------- | -------------- | --------------- |
| BulkCreateFiles_bulk_1000_files_medium | 12.2x faster 🚀 | 79.1x faster 🚀  |
| BulkCreateFiles_bulk_1000_files_small  | 11.4x faster 🚀 | 22.7x faster 🚀  |
| BulkCreateFiles_bulk_100_files_medium  | 21.5x faster 🚀 | 63.8x faster 🚀  |
| BulkCreateFiles_bulk_100_files_small   | 17.1x faster 🚀 | 8.5x faster 🚀   |
| CompareCommits_adjacent_commits_large  | 10.6x faster 🚀 | 22.2x faster 🚀  |
| CompareCommits_adjacent_commits_medium | 6.4x faster 🚀  | 4.3x faster 🚀   |
| CompareCommits_adjacent_commits_small  | 6.5x faster 🚀  | ~same ⚖️         |
| CompareCommits_adjacent_commits_xlarge | 32.4x faster 🚀 | 121.9x faster 🚀 |
| CompareCommits_few_commits_large       | 6.8x faster 🚀  | 13.7x faster 🚀  |
| CompareCommits_few_commits_medium      | 3.8x faster 🚀  | 2.3x faster 🚀   |
| CompareCommits_few_commits_small       | 3.3x faster 🚀  | 1.8x slower 🐌   |
| CompareCommits_few_commits_xlarge      | 21.8x faster 🚀 | 77.1x faster 🚀  |
| CompareCommits_max_commits_large       | 4.3x faster 🚀  | 8.5x faster 🚀   |
| CompareCommits_max_commits_medium      | 2.5x faster 🚀  | 1.8x faster ✅   |
| CompareCommits_max_commits_small       | 2.2x faster 🚀  | 2.6x slower 🐌   |
| CompareCommits_max_commits_xlarge      | 15.9x faster 🚀 | 55.3x faster 🚀  |
| CreateFile_large_repo                  | 32.3x faster 🚀 | 46.6x faster 🚀  |
| CreateFile_medium_repo                 | 25.2x faster 🚀 | 9.6x faster 🚀   |
| CreateFile_small_repo                  | 19.8x faster 🚀 | 2.0x faster 🚀   |
| CreateFile_xlarge_repo                 | 75.2x faster 🚀 | 258.0x faster 🚀 |
| DeleteFile_large_repo                  | 34.8x faster 🚀 | 48.3x faster 🚀  |
| DeleteFile_medium_repo                 | 26.4x faster 🚀 | 9.7x faster 🚀   |
| DeleteFile_small_repo                  | 22.3x faster 🚀 | 2.2x faster 🚀   |
| DeleteFile_xlarge_repo                 | 80.1x faster 🚀 | 277.9x faster 🚀 |
| GetFlatTree_large_tree                 | 25.2x faster 🚀 | 49.3x faster 🚀  |
| GetFlatTree_medium_tree                | 18.0x faster 🚀 | 11.2x faster 🚀  |
| GetFlatTree_small_tree                 | 10.4x faster 🚀 | 1.5x faster ✅   |
| GetFlatTree_xlarge_tree                | 68.3x faster 🚀 | 252.7x faster 🚀 |
| UpdateFile_large_repo                  | 28.1x faster 🚀 | 40.1x faster 🚀  |
| UpdateFile_medium_repo                 | 25.9x faster 🚀 | 9.5x faster 🚀   |
| UpdateFile_small_repo                  | 24.1x faster 🚀 | 2.4x faster 🚀   |
| UpdateFile_xlarge_repo                 | 83.4x faster 🚀 | 276.4x faster 🚀 |

### 💾 Memory Comparison

*Note: git-cli uses minimal memory as it stores data on disk, not in memory*

| Operation                              | vs git-cli    | vs go-git     |
| -------------------------------------- | ------------- | ------------- |
| BulkCreateFiles_bulk_1000_files_medium | -1.4x more 💾  | 8.0x less 💚   |
| BulkCreateFiles_bulk_1000_files_small  | -6.0x more 💾  | 2.6x less 💚   |
| BulkCreateFiles_bulk_100_files_medium  | 0.9x more 💾   | 12.3x less 💚  |
| BulkCreateFiles_bulk_100_files_small   | 0.7x more 💾   | 2.5x less 💚   |
| CompareCommits_adjacent_commits_large  | 97.7x more 💾  | 33.7x less 💚  |
| CompareCommits_adjacent_commits_medium | 40.3x more 💾  | 14.3x less 💚  |
| CompareCommits_adjacent_commits_small  | 20.4x more 💾  | 4.0x less 💚   |
| CompareCommits_adjacent_commits_xlarge | 254.0x more 💾 | 84.2x less 💚  |
| CompareCommits_few_commits_large       | 100.8x more 💾 | 25.7x less 💚  |
| CompareCommits_few_commits_medium      | 50.8x more 💾  | 12.4x less 💚  |
| CompareCommits_few_commits_small       | 30.9x more 💾  | 3.1x less 💚   |
| CompareCommits_few_commits_xlarge      | 254.0x more 💾 | 92.3x less 💚  |
| CompareCommits_max_commits_large       | 107.3x more 💾 | 30.2x less 💚  |
| CompareCommits_max_commits_medium      | 61.6x more 💾  | 10.0x less 💚  |
| CompareCommits_max_commits_small       | 42.2x more 💾  | 2.5x less 💚   |
| CompareCommits_max_commits_xlarge      | 266.0x more 💾 | 88.8x less 💚  |
| CreateFile_large_repo                  | 28.2x more 💾  | 73.1x less 💚  |
| CreateFile_medium_repo                 | 11.3x more 💾  | 22.6x less 💚  |
| CreateFile_small_repo                  | 11.5x more 💾  | 2.9x less 💚   |
| CreateFile_xlarge_repo                 | 94.9x more 💾  | 157.5x less 💚 |
| DeleteFile_large_repo                  | 25.3x more 💾  | 81.3x less 💚  |
| DeleteFile_medium_repo                 | 10.4x more 💾  | 32.0x less 💚  |
| DeleteFile_small_repo                  | 10.9x more 💾  | 2.6x less 💚   |
| DeleteFile_xlarge_repo                 | 95.9x more 💾  | 159.0x less 💚 |
| GetFlatTree_large_tree                 | 1.2x more 💾   | 60.7x less 💚  |
| GetFlatTree_medium_tree                | 1.8x more 💾   | 33.4x less 💚  |
| GetFlatTree_small_tree                 | 4.4x more 💾   | 6.3x less 💚   |
| GetFlatTree_xlarge_tree                | 0.6x more 💾   | 141.4x less 💚 |
| UpdateFile_large_repo                  | 27.4x more 💾  | 75.2x less 💚  |
| UpdateFile_medium_repo                 | 10.5x more 💾  | 20.0x less 💚  |
| UpdateFile_small_repo                  | 10.4x more 💾  | 2.9x less 💚   |
| UpdateFile_xlarge_repo                 | 97.0x more 💾  | 157.3x less 💚 |

## 📈 Detailed Statistics

### BulkCreateFiles_bulk_1000_files_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 10.93s       | 10.93s       | -5946528 B | -5946528 B    |
| go-git  | 1    | ⚠️ 0.0%   | 70.61s       | 70.61s       | 61.0 MB    | 61.0 MB       |
| nanogit | 1    | ✅ 100.0% | 892.4ms      | 892.4ms      | 7.7 MB     | 7.7 MB        |

### BulkCreateFiles_bulk_1000_files_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 9.66s        | 9.66s        | -1048968 B | -1048968 B    |
| go-git  | 1    | ⚠️ 0.0%   | 19.25s       | 19.25s       | 15.4 MB    | 15.4 MB       |
| nanogit | 1    | ✅ 100.0% | 846.5ms      | 846.5ms      | 6.0 MB     | 6.0 MB        |

### BulkCreateFiles_bulk_100_files_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 2.15s        | 2.15s        | 5.1 MB     | 5.1 MB        |
| go-git  | 1    | ⚠️ 0.0%   | 6.38s        | 6.38s        | 54.1 MB    | 54.1 MB       |
| nanogit | 1    | ✅ 100.0% | 99.9ms       | 99.9ms       | 4.4 MB     | 4.4 MB        |

### BulkCreateFiles_bulk_100_files_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.60s        | 1.60s        | 5.1 MB     | 5.1 MB        |
| go-git  | 1    | ⚠️ 0.0%   | 798.8ms      | 798.8ms      | 8.9 MB     | 8.9 MB        |
| nanogit | 1    | ✅ 100.0% | 93.6ms       | 93.6ms       | 3.5 MB     | 3.5 MB        |

### CompareCommits_adjacent_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.26s        | 1.26s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.64s        | 2.64s        | 227.1 MB   | 227.1 MB      |
| nanogit | 1    | ✅ 100.0% | 118.8ms      | 118.8ms      | 6.7 MB     | 6.7 MB        |

### CompareCommits_adjacent_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 659.3ms      | 659.3ms      | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 442.2ms      | 442.2ms      | 39.6 MB    | 39.6 MB       |
| nanogit | 1    | ✅ 100.0% | 103.2ms      | 103.2ms      | 2.8 MB     | 2.8 MB        |

### CompareCommits_adjacent_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 528.6ms      | 528.6ms      | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 74.3ms       | 74.3ms       | 5.7 MB     | 5.7 MB        |
| nanogit | 1    | ✅ 100.0% | 81.0ms       | 81.0ms       | 1.4 MB     | 1.4 MB        |

### CompareCommits_adjacent_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.37s        | 5.37s        | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.17s       | 20.17s       | 1.4 GB     | 1.4 GB        |
| nanogit | 1    | ✅ 100.0% | 165.6ms      | 165.6ms      | 17.4 MB    | 17.4 MB       |

### CompareCommits_few_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.28s        | 1.28s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.57s        | 2.57s        | 178.5 MB   | 178.5 MB      |
| nanogit | 1    | ✅ 100.0% | 188.4ms      | 188.4ms      | 6.9 MB     | 6.9 MB        |

### CompareCommits_few_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 655.4ms      | 655.4ms      | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 400.1ms      | 400.1ms      | 43.2 MB    | 43.2 MB       |
| nanogit | 1    | ✅ 100.0% | 174.4ms      | 174.4ms      | 3.5 MB     | 3.5 MB        |

### CompareCommits_few_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 523.1ms      | 523.1ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 86.8ms       | 86.8ms       | 6.5 MB     | 6.5 MB        |
| nanogit | 1    | ✅ 100.0% | 158.5ms      | 158.5ms      | 2.1 MB     | 2.1 MB        |

### CompareCommits_few_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.68s        | 5.68s        | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.10s       | 20.10s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 260.7ms      | 260.7ms      | 17.4 MB    | 17.4 MB       |

### CompareCommits_max_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.29s        | 1.29s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.58s        | 2.58s        | 223.1 MB   | 223.1 MB      |
| nanogit | 1    | ✅ 100.0% | 301.8ms      | 301.8ms      | 7.4 MB     | 7.4 MB        |

### CompareCommits_max_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 670.8ms      | 670.8ms      | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 468.2ms      | 468.2ms      | 42.2 MB    | 42.2 MB       |
| nanogit | 1    | ✅ 100.0% | 264.3ms      | 264.3ms      | 4.2 MB     | 4.2 MB        |

### CompareCommits_max_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 529.3ms      | 529.3ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 94.0ms       | 94.0ms       | 7.2 MB     | 7.2 MB        |
| nanogit | 1    | ✅ 100.0% | 244.8ms      | 244.8ms      | 2.9 MB     | 2.9 MB        |

### CompareCommits_max_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.80s        | 5.80s        | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.17s       | 20.17s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 364.6ms      | 364.6ms      | 18.2 MB    | 18.2 MB       |

### CreateFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.99s        | 2.01s        | 135.7 KB   | 135.6 KB      |
| go-git  | 3    | ✅ 100.0% | 2.86s        | 2.88s        | 273.0 MB   | 274.7 MB      |
| nanogit | 3    | ✅ 100.0% | 61.5ms       | 64.3ms       | 3.7 MB     | 3.8 MB        |

### CreateFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.35s        | 1.38s        | 136.7 KB   | 135.9 KB      |
| go-git  | 3    | ✅ 100.0% | 516.2ms      | 526.4ms      | 33.9 MB    | 37.9 MB       |
| nanogit | 3    | ✅ 100.0% | 53.7ms       | 59.5ms       | 1.5 MB     | 1.4 MB        |

### CreateFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.18s        | 1.20s        | 136.3 KB   | 136.1 KB      |
| go-git  | 3    | ✅ 100.0% | 121.6ms      | 142.1ms      | 4.4 MB     | 4.3 MB        |
| nanogit | 3    | ✅ 100.0% | 59.6ms       | 80.3ms       | 1.5 MB     | 1.6 MB        |

### CreateFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 6.49s        | 6.71s        | 135.8 KB   | 135.9 KB      |
| go-git  | 3    | ✅ 100.0% | 22.27s       | 22.43s       | 1.9 GB     | 1.9 GB        |
| nanogit | 3    | ✅ 100.0% | 86.3ms       | 96.7ms       | 12.6 MB    | 12.6 MB       |

### DeleteFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.05s        | 2.10s        | 135.8 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 2.85s        | 2.88s        | 273.0 MB   | 272.8 MB      |
| nanogit | 3    | ✅ 100.0% | 59.0ms       | 66.0ms       | 3.4 MB     | 3.5 MB        |

### DeleteFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.34s        | 1.37s        | 135.7 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 495.5ms      | 507.8ms      | 44.3 MB    | 45.5 MB       |
| nanogit | 3    | ✅ 100.0% | 51.0ms       | 53.4ms       | 1.4 MB     | 1.4 MB        |

### DeleteFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.16s        | 1.16s        | 136.1 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 116.2ms      | 130.2ms      | 3.8 MB     | 4.0 MB        |
| nanogit | 3    | ✅ 100.0% | 52.0ms       | 57.3ms       | 1.4 MB     | 1.5 MB        |

### DeleteFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 6.50s        | 6.83s        | 135.6 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 22.54s       | 22.89s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 81.1ms       | 83.4ms       | 12.7 MB    | 12.8 MB       |

### GetFlatTree_large_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.34s        | 1.34s        | 3.2 MB     | 3.2 MB        |
| go-git  | 1    | ✅ 100.0% | 2.61s        | 2.61s        | 241.1 MB   | 241.1 MB      |
| nanogit | 1    | ✅ 100.0% | 52.9ms       | 52.9ms       | 4.0 MB     | 4.0 MB        |

### GetFlatTree_medium_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 713.2ms      | 713.2ms      | 740.5 KB   | 740.5 KB      |
| go-git  | 1    | ✅ 100.0% | 443.7ms      | 443.7ms      | 44.4 MB    | 44.4 MB       |
| nanogit | 1    | ✅ 100.0% | 39.6ms       | 39.6ms       | 1.3 MB     | 1.3 MB        |

### GetFlatTree_small_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 542.9ms      | 542.9ms      | 154.6 KB   | 154.6 KB      |
| go-git  | 1    | ✅ 100.0% | 75.9ms       | 75.9ms       | 4.2 MB     | 4.2 MB        |
| nanogit | 1    | ✅ 100.0% | 52.2ms       | 52.2ms       | 677.2 KB   | 677.2 KB      |

### GetFlatTree_xlarge_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.36s        | 5.36s        | 18.7 MB    | 18.7 MB       |
| go-git  | 1    | ✅ 100.0% | 19.84s       | 19.84s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 78.5ms       | 78.5ms       | 11.3 MB    | 11.3 MB       |

### UpdateFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.00s        | 2.02s        | 135.2 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 2.86s        | 2.87s        | 271.9 MB   | 271.8 MB      |
| nanogit | 3    | ✅ 100.0% | 71.2ms       | 101.7ms      | 3.6 MB     | 3.7 MB        |

### UpdateFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.36s        | 1.38s        | 135.5 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 499.0ms      | 502.8ms      | 27.9 MB    | 25.3 MB       |
| nanogit | 3    | ✅ 100.0% | 52.6ms       | 53.2ms       | 1.4 MB     | 1.4 MB        |

### UpdateFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.16s        | 1.17s        | 135.2 KB   | 135.2 KB      |
| go-git  | 3    | ✅ 100.0% | 113.9ms      | 122.5ms      | 4.0 MB     | 4.0 MB        |
| nanogit | 3    | ✅ 100.0% | 48.3ms       | 52.2ms       | 1.4 MB     | 1.4 MB        |

### UpdateFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 6.78s        | 7.23s        | 135.1 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 22.46s       | 23.18s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 81.3ms       | 85.2ms       | 12.8 MB    | 12.7 MB       |

