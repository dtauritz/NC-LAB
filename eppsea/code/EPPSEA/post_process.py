import statistics
import sys
import pickle
import re

import numpy as np

from statsmodels.stats.proportion import proportions_ztest
import scipy.stats

import matplotlib
matplotlib.use('Agg')
from matplotlib import pyplot as plt

def get_eval_counts(results):
    all_eval_counts = []
    for r in results:
        all_eval_counts.extend(r['eval_counts'])
    return sorted(list(set(all_eval_counts)))

def main(final_output_directory, results_file_paths):
    # first, load all results
    results = []
    for fp in results_file_paths:
        with open(fp, 'rb') as file:
            results.extend(pickle.load(file))

    # get the fitness function names and ids
    fitness_function_ids = []
    fitness_function_display_names = dict()

    for r in results:
        if r['fitness_function_id'] not in fitness_function_ids:
            fitness_function_ids.append(r['fitness_function_id'])
            fitness_function_display_names[r['fitness_function_id']] = r['fitness_function_display_name']

    # get the selection function names and ids
    selection_function_ids = []
    selection_function_display_names = dict()

    for r in results:
        if r['selection_function_id'] not in selection_function_ids:
            selection_function_ids.append(r['selection_function_id'])
            selection_function_display_names[r['selection_function_id']] = r['selection_function_display_name']

    np.seterr(all='warn')
    # log string forms of eppsea-based selection functions
    printed_selection_functions = []
    for r in results:
        if r['selection_function_eppsea_string'] not in printed_selection_functions:
            print('String form of {0}: {1}'.format(r['selection_function_display_name'], r['selection_function_eppsea_string']))
            printed_selection_functions.append(r['selection_function_eppsea_string'])
    # Analyze results for each fitness function
    num_targets_hit = [0]*len(selection_function_ids);
    num_tests_run = [0]*len(selection_function_ids);
    functionClassHitPercentage = {}
    functionClassRegex = re.compile('F\d{1,2}')
    for fitness_function_id in fitness_function_ids:
        plt.clf()
        print('--------------------------- Analyzing results for fitness function with id {0} ---------------------------------'.format(fitness_function_id))
        # Get the name of the fitness function from one of the result files
        fitness_function_name = fitness_function_display_names[fitness_function_id]
        functionClass = functionClassRegex.search(fitness_function_name).group(0)
        if functionClass not in functionClassHitPercentage:
            functionClassHitPercentage[functionClass] = [0]*len(selection_function_ids)
        print(functionClass)
        print('Fitness Function Name: ' + fitness_function_name)
        print('Plotting figure')


        # filter out the results for this fitness function
        fitness_function_results = list(r for r in results if r['fitness_function_id'] == fitness_function_id)

        # Set the plot to use Log Scale
        plt.yscale('symlog')

        # Plot results for each selection function
        for selection_function_id in selection_function_ids:
            selection_function_results = list(r for r in fitness_function_results if r['selection_function_id'] == selection_function_id)
            selection_function_name = selection_function_display_names[selection_function_id]
            mu = get_eval_counts(selection_function_results)
            average_best_fitnesses = []
            for m in mu:
                average_best_fitnesses.append(statistics.mean(
                    r['best_fitnesses'][m] for r in selection_function_results if m in r['best_fitnesses']))

            plt.plot(mu, average_best_fitnesses, label=selection_function_name)

        plt.xlabel('Evaluations')
        plt.ylabel('Best Fitness')
        plt.legend(loc=(1.02, 0))
        plt.savefig('{0}/figure_{1}.png'.format(final_output_directory, fitness_function_id),
                    bbox_inches='tight')

        print('Plotting boxplot')
        final_best_fitnesses_list = []
        selection_name_list = []

        # Set the plot to use Log Scale
        plt.yscale('symlog')

        for selection_function_id in selection_function_ids:
            selection_function_results = list(r for r in fitness_function_results if r['selection_function_id'] == selection_function_id)
            selection_function_name = selection_function_display_names[selection_function_id]
            selection_name_list.append(selection_function_name)
            final_best_fitnesses = list(r['final_best_fitness'] for r in selection_function_results)
            final_best_fitnesses_list.append(final_best_fitnesses)
        plt.boxplot(final_best_fitnesses_list, labels=selection_name_list)

        plt.xlabel('Evaluations')
        plt.xticks(rotation=90)
        plt.ylabel('Final Best Fitness')
        legend = plt.legend([])
        legend.remove()
        plt.savefig('{0}/boxplot_{1}.png'.format(final_output_directory, fitness_function_id),
                    bbox_inches='tight')

        print('Doing t-tests')

        tested_pairs = []
        significant_differences = []
        counter = 0
        for selection_function_id1 in selection_function_ids:
            selection_function_results1 = list(r for r in fitness_function_results if r['selection_function_id'] == selection_function_id1)
            selection_function_target_results1 = list(r for r in selection_function_results1 if r['termination_reason'] == 'target_fitness_hit')
            selection_function_name1 = selection_function_display_names[selection_function_id1]
            final_best_fitnesses1 = list(r['final_best_fitness'] for r in selection_function_results1)
            # round means to 5 decimal places for cleaner display
            average_final_best_fitness1 = round(statistics.mean(final_best_fitnesses1), 5)
            target_hit_percentage1 = round(len(selection_function_target_results1) * 100 / len(selection_function_results1), 2)
            num_targets_hit[counter] = num_targets_hit[counter] + len(selection_function_target_results1)
            num_tests_run[counter] = num_tests_run[counter] + len(selection_function_results1)

            functionClassHitPercentage[functionClass][counter] = functionClassHitPercentage[functionClass][counter] + target_hit_percentage1
            counter = counter + 1
            print('Mean performance of {0}: {1}, reaching target fitness in {2}% of runs'.format(selection_function_name1,average_final_best_fitness1, target_hit_percentage1))
            # perform a t test with all the other results that this selection has not yet been tested against
            for selection_function_id2 in selection_function_ids:
                if selection_function_id2 != selection_function_id1 and (selection_function_id1, selection_function_id2) not in tested_pairs and (selection_function_id2, selection_function_id1) not in tested_pairs:
                    selection_function_results2 = list(r for r in fitness_function_results if r['selection_function_id'] == selection_function_id2)
                    selection_function_target_results2 = list(r for r in selection_function_results2 if r['termination_reason'] == 'target_fitness_hit')
                    selection_function_name2 = selection_function_display_names[selection_function_id2]
                    final_best_fitnesses2 = list(r['final_best_fitness'] for r in selection_function_results2)
                    # round means to 5 decimal places for cleaner display
                    average_final_best_fitness2 = round(statistics.mean(final_best_fitnesses2), 5)
                    target_hit_percentage2 = round(len(selection_function_target_results2) * 100 / len(selection_function_results2), 2)

                    _, p_fitness = scipy.stats.ttest_rel(final_best_fitnesses1, final_best_fitnesses2)
                    mean_difference_fitness = round(average_final_best_fitness1 - average_final_best_fitness2, 5)

                    if p_fitness < 0.05:
                        significant_differences.append((selection_function_name1, selection_function_name2, mean_difference_fitness, p_fitness))

                    #final_target_evals1 = list(max(r['eval_counts']) for r in selection_function_target_results1)
                    #final_target_evals2 = list(max(r['eval_counts']) for r in selection_function_target_results2)

                    #if len(final_target_evals1) > 0 and len(final_target_evals2) > 0:
                    #    mean_difference_evals = round(statistics.mean(final_target_evals1) - statistics.mean(final_target_evals2), 5)

                    #    if mean_difference_evals < 0:
                    #        print('\t\t{0} used {1} fewer evals to hit target fitness'.format(selection_function_name1,mean_difference_evals))
                    #    else:
                    #        print('\t\t{0} used {1} more evals to hit target fitness'.format(selection_function_name1,mean_difference_evals))

                    tested_pairs.append((selection_function_id1, selection_function_id2))

        if significant_differences:
            for selection_function_name1, selection_function_name2, mean_difference_fitness, p_fitness in significant_differences:
                if mean_difference_fitness > 0:
                    print('\t{0} performed {1} higher than {2}, p={3}'.format(selection_function_name1, mean_difference_fitness, selection_function_name2, p_fitness))
                else:
                    print('\t{0} performed {1} lower than {2}, p={3}'.format(selection_function_name1, mean_difference_fitness, selection_function_name2, p_fitness))
        else:
            print('\tNo significant differences in performance')

    for i in range(0,len(selection_function_ids[:-1])):
        print('Selection function {0} hit target at an average of {1}%'.format(i, 100 * num_targets_hit[i]/num_tests_run[i]))
    print('Basic CMAES hit target at an average of {0}%'.format(100 * num_targets_hit[-1]/num_tests_run[-1]))
    print()

    for i in range(0,len(selection_function_ids[:-1])):
        mean_difference_prop = round(num_targets_hit[i]/num_tests_run[i] - num_targets_hit[-1]/num_tests_run[-1], 5)
        if mean_difference_prop is not 0:
            successes = [num_targets_hit[i], num_targets_hit[-1]]
            trials = [num_tests_run[i], num_tests_run[-1]]
            _, p_fitness = proportions_ztest(successes, trials)
            if p_fitness < 0.05:
                if mean_difference_prop > 0:
                    print('\tSelection function {0} performed {1}% better than Basic CMAES, p={2}'.format(i, mean_difference_prop*100, p_fitness))
                else:
                    print('\tSelection function {0} performed {1}% worse than Basic CMAES, p={2}'.format(i, mean_difference_prop*-100, p_fitness))
            else:
                print('\tSelection function {0} performed statistically the same as BASIC CMAES, p={1}'.format(i,p_fitness))

    print()
    for funcClass in functionClassHitPercentage:
        for i in range(0,len(selection_function_ids)):
            print('Selection function {0} hit target at an average of {1}% in class {2}'.format(i, len(functionClassHitPercentage) * functionClassHitPercentage[funcClass][i]/len(fitness_function_ids), funcClass))

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print('Please provide output directory and results file(s)')
        exit(1)
    final_output_directory = sys.argv[1]
    results_file_paths = sys.argv[2:]
    main(final_output_directory, results_file_paths)
