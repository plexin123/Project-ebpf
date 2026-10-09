*** 7 October 2026

- Problem:
    - Basically whenever we were only receiving the
      time from the server, we are cannot determine the self time of each
     function.
    - Example:
        - If we have a function A that calls function B, and function B takes 10ms to execute, and function A takes 15ms to execute, we cannot determine how much time was spent in function A itself (self time) versus the time spent in function B (child time).
        - This is because the server only provides the total time for function A, which includes the time spent in function B.
        - Visualize:
            - Function A: 15ms (total time)
                - Function B: 10ms (child time)
                - Self time of Function A: 5ms (total time - child time)

    