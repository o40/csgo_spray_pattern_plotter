from collections import defaultdict
from pathlib import Path
from recordtype import recordtype
import argparse
import collections
import itertools
import math
import matplotlib.pyplot as plt
import sys

plt.style.use('Solarize_Light2')

Points = recordtype('Points', ['x', 'y'])
Spray = recordtype('Spray', ['player', 'weapon', 'first_shot_tick', 'view_angles', 'shots', 'hits', 'kills'])


def weapon_id_to_string(id):
    weapons = {
        301: "Galil",
        302: "Famas",
        303: "AK47",
        304: "M4A4",
        305: "M4A1",
        306: "Scout",
        307: "SG553",
        308: "AUG",
        309: "AWP",
        310: "Scar20",
        311: "G3SG1"
    }
    return weapons.get(id, "Unknown")


def should_adjust_horizontal_angle(points):
    '''
    Adjust horizontal angles to avoid plotting over the 360 -> 0 gap
    '''
    horizontal_min = min(points)
    horizontal_max = max(points)
    return horizontal_min < 20 and horizontal_max > 360 - 20


def adjust_horizontal_angles(angles):
    return [((angle + 180) % 360) for angle in angles]


def adjust_vertical_angle(angle):
    '''
    Vertical angles seem to be stored 270 -> 360 from down to "forward", and then
    0 -> 90 from "forward" to up. Adjust this to be able to plot.
    '''
    return (angle - 360) if angle > 180 else angle


def plot_line(axes, points):
    '''
    Plot the points as line segments (with alternating colors)
    '''
    line_color_toggle = False
    for x1, x2, y1, y2 in zip(points.x[:-1], points.x[1:], points.y[:-1], points.y[1:]):

        # Plot a point if no movement
        if x1 == x2 and y1 == y2:
            axes.scatter(x1, y1, c="white", alpha=1, zorder=2, s=4)
            continue

        line_color = 'darkgrey'
        if line_color_toggle:
            line_color = 'lightgrey'

        line_color_toggle ^= True
        axes.plot([x1, x2], [y1, y2], '-', c=line_color, linewidth=2, zorder=1)


def plot_shots(axes, shots, hits, kills):
    '''
    Scatter plot the shots and add labels (numbering) for them
    '''
    marker_size = 80
    axes.scatter(shots.x, shots.y, c="goldenrod", alpha=1, zorder=2, s=marker_size)
    axes.scatter(hits.x, hits.y, c="lime", alpha=1, zorder=2, s=marker_size)
    axes.scatter(kills.x, kills.y, c="red", alpha=1, zorder=2, s=marker_size)

    shotnum = 1
    for x, y in zip(shots.x, shots.y):
        axes.text(x, y, f'{shotnum}',
                  horizontalalignment='center',
                  verticalalignment='center',
                  fontsize=8, alpha=1, zorder=3)
        shotnum += 1


def plot_spray(spray,
               csv_file,
               output_folder):

    fig, axes = plt.subplots(1, 1, figsize=(10, 8))

    # Adjust angles for plotting purposes
    if should_adjust_horizontal_angle(spray.view_angles.x):
        spray.view_angles.x = adjust_horizontal_angles(spray.view_angles.x)
        spray.shots.x = adjust_horizontal_angles(spray.shots.x)
        spray.hits.x = adjust_horizontal_angles(spray.hits.x)
        spray.kills.x = adjust_horizontal_angles(spray.kills.x)

    # Plot the view angles as a line
    plot_line(axes, spray.view_angles)

    # Plot the shots as numbered points
    plot_shots(axes, spray.shots, spray.hits, spray.kills)

    weapon_str = weapon_id_to_string(spray.weapon)
    replay_name = Path(csv_file).stem
    axes.set_title(f"Game: {replay_name}.dem\n"
                   f"Tick: {spray.first_shot_tick}, Player: {spray.player}, Weapon: {weapon_str}")
    axes.set_xlabel("yaw (degrees)")
    axes.set_ylabel("pitch (degrees)")

    # CS:GO left and up is negative direction. Invert to plot correctly.
    axes.invert_xaxis()
    axes.invert_yaxis()

    filename = f"{output_folder}/{replay_name}_{spray.player}_{str(spray.first_shot_tick).zfill(7)}.png"
    plt.savefig(filename, facecolor=fig.get_facecolor())
    plt.close()


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--csv", required=True)
    parser.add_argument("--filter", required=False)
    parser.add_argument("--test", required=False, action='store_true')
    parser.add_argument("--tick", required=False)
    parser.add_argument("--out", required=False, default='out')
    return parser.parse_args()


def main():

    args = parse_args()

    # Settings
    segment_margin = 20
    min_shots_for_plot = 4

    sprays = []

    prev_tick = None

    with open(args.csv) as rawfile:

        spray = Spray(None, None, None, Points([], []), Points([], []), Points([], []), Points([], []))

        # Add view angles in chunks to avoid view angles to be plotted after last shot
        view_angles_temp = Points([], [])

        for index, row in enumerate(rawfile):
            player, tick, shot, hit, kill, x, y, weapon = row.split(',')
            tick, shot, hit, kill, weapon = int(tick), int(shot), int(hit), int(kill), int(weapon)
            x, y = float(x), float(y)

            y = adjust_vertical_angle(y)

            if args.filter and args.filter != player:
                continue

            # Check in jump of tick. In that case store spray and continue
            if prev_tick:
                if abs(tick - prev_tick) > segment_margin:
                    sprays.append(spray)
                    spray = Spray(None, None, None, Points([], []), Points([], []), Points([], []), Points([], []))
                    view_angles_temp = Points([], [])
            prev_tick = tick

            view_angles_temp.x.append(x)
            view_angles_temp.y.append(y)

            if shot:
                spray.view_angles.x.extend(view_angles_temp.x)
                spray.view_angles.y.extend(view_angles_temp.y)
                view_angles_temp = Points([], [])

                if spray.player is None:
                    spray.player = player

                if spray.weapon is None:
                    spray.weapon = weapon

                if spray.first_shot_tick is None:
                    spray.first_shot_tick = tick

                spray.shots.x.append(x)
                spray.shots.y.append(y)

                if hit:
                    spray.hits.x.append(x)
                    spray.hits.y.append(y)

                if kill:
                    spray.kills.x.append(x)
                    spray.kills.y.append(y)
        sprays.append(spray)

        for spray in sprays:
            if args.tick and (int(args.tick) != spray.first_shot_tick):
                continue
            if len(spray.shots.x) >= min_shots_for_plot:
                plot_spray(spray, args.csv, args.out)
                if args.test:
                    print("Returning early for testing")
                    sys.exit(0)


if __name__ == '__main__':
    main()
